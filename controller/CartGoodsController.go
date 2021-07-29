package controller

import (
	"TradePlatform/common"
	"TradePlatform/model"
	"TradePlatform/response"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unsafe"
)

type Data struct {
	Name string 		`json:"name"`
	Price float64 		`json:"price"`
	Kind int 			`json:"type"`
	MainPicture string	`json:"mainPicture"`
	MorePicture string	`json:"morePicture"`
	WordDetail string	`json:"wordDetail"`
	Email string		`json:"ownerEmail"`
}

type Output struct {
	RetCode int		`json:"code"`
	RetData Data	`json:"data"`
	RetMsg string	`json:"msg"`
}

func AddCart (ctx *gin.Context) {
	var requestCart model.Cart
	if err := ctx.Bind(&requestCart); err != nil {
		log.Println("AddCart参数错误")
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	if requestCart.GoodsID == 0 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "参数缺失")
		log.Println("goodsid缺失, " ,requestCart.GoodsID)
		return
	}

	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" || len(tokenString) < 7 {
		response.Response(ctx, 422, 422, nil, "参数缺失")
		return
	}
	if strings.HasPrefix(tokenString,"Bearer ") {
		tokenString = tokenString[7:]
	}
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
		return
	}

	requestCart.OwnerEmail = claims.UserEmail

	c := common.GetRedis()
	defer c.Close()
	script := "local t = redis.call('EXISTS', KEYS[1])\nif t == 0 then\n    redis.call('SET', KEYS[1], 1)\n    redis.call('PEXPIRE', KEYS[1], 200) -- 过期时间的设定需要考虑实际业务速度\n    return 1\nend\nreturn 0"
	if flag, err := redis.Int(c.Do("EVAL", script, "1", "cartLock" + requestCart.OwnerEmail)) ; err != nil || flag != 1  {
		log.Println("获取分布式锁失败, err = ", err)
		response.Response(ctx, 503, 503, nil, "服务器错误")
		return
	}

	// 通过/api/goods/detail 获取商品所有信息
	song := make(map[string]string)
	song["id"] = strconv.Itoa(int(requestCart.GoodsID))
	song["email"] = requestCart.OwnerEmail
	bytesData, _ := json.Marshal(song)

	res, err := http.Post("http://152.136.180.243:8000/api/goods/detail", // TODO xutianmeng
		"application/json;charset=utf-8", bytes.NewBuffer(bytesData))
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
	}

	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
	}
	str := (*string)(unsafe.Pointer(&content)) //转化为string,优化内存

	var temp Output

	_ = json.Unmarshal([]byte(*str), &temp)

	requestCart.Name = temp.RetData.Name
	requestCart.Price = temp.RetData.Price
	requestCart.MainPicture = temp.RetData.MainPicture

	log.Println(*str)
	log.Println("temp = ",temp)
	log.Println("requestCart = ",requestCart)

	response.Success(ctx, gin.H{}, "加入购物车成功")

	go func (localCart model.Cart) {
		model.CartChannel <- struct {} {}
		defer func () {
			<- model.CartChannel
		} ()
		db := common.GetDB()
		c := common.GetRedis()
		defer c.Close()
		newCart := model.Cart {
			Name: localCart.Name,
			GoodsID: localCart.GoodsID,
			OwnerEmail: localCart.OwnerEmail,
			Price: localCart.Price,
			MainPicture: localCart.MainPicture,
		}
		err := db.Create(&newCart).Error
		if err != nil {
			log.Println("插入mysql cart数据失败")
		}
		// redis
		_, err = c.Do("LPUSH", "Cart:" + requestCart.OwnerEmail, localCart.GoodsID)
		if err != nil {
			log.Println("add Cart redis failed。 err = ", err)
		}
		script = "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
		_, err = c.Do("EVAL", script, "1", "cartLock" + localCart.OwnerEmail)
		if err != nil {
			log.Println("释放分布式锁失败, OwnerEmail = ", localCart.OwnerEmail, ", err = ", err.Error())
		}
	} (requestCart)
}

// post email + goodsID
func DelCart (ctx *gin.Context) {
	var requestCart model.Cart
	if err := ctx.Bind(&requestCart); err != nil {
		log.Println("DelCart参数错误")
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	log.Println("delcart, ", requestCart)

	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" || len(tokenString) < 7 {
		response.Response(ctx, 422, 422, nil, "参数缺失")
		return
	}
	if strings.HasPrefix(tokenString,"Bearer ") {
		tokenString = tokenString[7:]
	}
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid {
		if err == nil {
			err = errors.New("token not valid")
		}
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
		return
	}

	requestCart.OwnerEmail = claims.UserEmail

	// 获取锁
	c := common.GetRedis()
	defer c.Close()
	script := "local t = redis.call('EXISTS', KEYS[1])\nif t == 0 then\n    redis.call('SET', KEYS[1], 1)\n    redis.call('PEXPIRE', KEYS[1], 200) -- 过期时间的设定需要考虑实际业务速度\n    return 1\nend\nreturn 0"
	if flag, err := redis.Int(c.Do("EVAL", script, "1", "cartLock" + requestCart.OwnerEmail)) ; err != nil || flag != 1  {
		log.Println("获取分布式锁失败, err = ", err)
		response.Response(ctx, 503, 503, nil, "服务器错误")
		return
	}

	response.Success(ctx, gin.H{}, "删除成功")

	go func (localCart model.Cart) {
		model.CartChannel <- struct {} {}
		defer func () {
			<- model.CartChannel
		} ()
		db := common.GetDB()
		c := common.GetRedis()
		defer c.Close()
		alterField := make(map[string]interface{})
		alterField["deleted"] = 1
		a := db.Model(&model.Cart{}).Where("owner_email = ? and goods_id = ? and deleted = 0", localCart.OwnerEmail, uint(localCart.GoodsID)).Update(alterField)
		log.Println("owner_email = ", localCart.OwnerEmail, " and goods_id = ", localCart.GoodsID, " and deleted = 0")
		if a.Error != nil {
			log.Println(a.Error)
		}
		// LREM key count value
		_, err := c.Do("LREM", "Cart:"+localCart.OwnerEmail, 0, localCart.GoodsID)
		//_, err = c.Do("DEL", "cart:"+requestCart.OwnerEmail)
		if err != nil {
			log.Println("del cart redis failed. err = ", err)
		}
		script = "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
		_, err = c.Do("EVAL", script, "1", "cartLock"+localCart.OwnerEmail)
		if err != nil {
			log.Println("释放分布式锁失败, OwnerEmail = ", localCart.OwnerEmail, ", err = ", err.Error())
		}
	} (requestCart)
}

func CartInfo (ctx *gin.Context) {
	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" || len(tokenString) < 7 {
		response.Response(ctx, 422, 422, nil, "参数缺失")
		return
	}
	if strings.HasPrefix(tokenString,"Bearer ") {
		tokenString = tokenString[7:]
	}
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
		return
	}

	var names []string
	var prices []float64
	var mainPictures []string
	var IDs []uint

	//script := "local key = \"Cart:\" .. KEYS[1]\nlocal ids = redis.call('LRANGE', key, 0, -1)\nlocal ret = {}\nfor i = 0, #ids do\n    local GoodsKey = \"GOodsID\" .. ids[i]\n    local name = redis.call('HGET', GoodsKey, 'name')\n    local price = redis.call('HGET', GoodsKey, 'price')\n    local mainPicture = redis.call('HGET', GoodsKey, 'mainPicture')\n    ret[#ret+1] = name..\",\"..price..\",\"..mainPicture\nend\nreturn ret"
	script := `
		-- 获取购物车信息 从队列中拿到goodsID再拿数据 KEYS[1] = ownerEmail
		local key = "Cart:" .. KEYS[1]
		local ids = redis.call('LRANGE', key, 0, -1)
		local ret = {}
		for i = 1, #ids do
			local GoodsKey = "GoodsID" .. ids[i]
			local name = redis.call('HGET', GoodsKey, 'name')
			local price = redis.call('HGET', GoodsKey, 'price')
			local mainPicture = redis.call('HGET', GoodsKey, 'mainPicture')
			ret[#ret+1] = ids[i]..","..name..","..price..","..mainPicture
		end
		return ret`
	c := common.GetRedis()
	defer c.Close()
	res, err := redis.Values(c.Do("EVAL", script, "1", claims.UserEmail))
	log.Println("CartInfo UserEmail = ",claims.UserEmail)
	if err != nil {
		log.Println("find cartInfo failed from redis. err = ", err)
		// find from mysql
		var tempCart []model.Cart
		db := common.GetDB()
		err := db.Select("goods_id, name, price, main_picture").Where("owner_email = ? and deleted = 0", claims.UserEmail).Find(&tempCart).Error
		if err != nil {
			log.Println("find cartInfo from mysql failed. err = ", err)
			response.Fail(ctx, "未查询到数据", gin.H{})
			return
		}
		for _, v := range tempCart {
			IDs = append(IDs, v.GoodsID)
			names = append(names, v.Name)
			prices = append(prices, v.Price)
			mainPictures = append(mainPictures, v.MainPicture)
		}
		response.Success(ctx, gin.H{
			"IDs" : IDs,
			"names" : names,
			"prices" : prices,
			"mainPictures" : mainPictures,
		}, "OK")
		return
	}
	for _, v := range res {
		if v == nil {
			continue
		}
		temp := strings.Split(string(v.([]byte)), ",")
		ID, _ := strconv.ParseUint(temp[0], 10, 64)
		IDs = append(IDs, uint(ID))
		names = append(names, temp[1])
		price, _ := strconv.ParseFloat(temp[2], 64)
		prices = append(prices, price)
		mainPictures = append(mainPictures, temp[3])
	}
	response.Success(ctx, gin.H{
		"IDs" : IDs,
		"names" : names,
		"prices" : prices,
		"mainPictures" : mainPictures,
	}, "OK")
}
