package controller

import (
	"TradePlatform/common"
	"TradePlatform/model"
	"TradePlatform/response"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	PICTUREDIR = "/home/ubuntu/workspace/GoLang/src/TradePlatform/goodsPicture/"
)

type GoodsRequest struct {
	GoodsType string `json:"GoodsType" form:"GoodsType"`
	Pn        string `json:"pn" form:"pn"`
	Rn        string `json:"rn" form:"rn"`
	Deleted   string `json:"deleted" form:"deleted"`
	Id        string `json:"id" form:"id"`
	Email     string `json:"email" form:"email"`
}

// 上传图片
func GoodsPictureUploads (ctx *gin.Context) {
	err := ctx.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Println(err)
		response.Response(ctx, 422, 422, nil, "上传失败")
	}
	//获取所有上传文件信息
	fhs := ctx.Request.MultipartForm.File["file"]
	log.Println("fhs = ", fhs)
	log.Println(ctx.Request.MultipartForm.Value)
	//ctx.FormFile()

	uuid, err := exec.Command("uuidgen").Output()
	if err != nil {
		log.Println(err)
		response.Response(ctx, 500, 500, nil, "服务器端错误")
		return
	}
	strUuid := string(uuid)
	strUuid = strUuid[0:8] + strUuid[9:13] + strUuid[14:18] + strUuid[19:23]
	uploadDir := PICTUREDIR + strUuid
	err = os.MkdirAll(uploadDir, 0777)
	if err != nil {
		response.Response(ctx, 422, 422, nil, "上传失败")
		log.Println(err)
		return
	}
	i := 0
	for _, fheader := range fhs {
		i++
		newFileName := strconv.Itoa(i) + ".png"
		if err := ctx.SaveUploadedFile(fheader, uploadDir+"/"+newFileName); err != nil {
			response.Response(ctx, 422, 422, nil, "上传失败")
			log.Println("上传错误：",err)
			return
		}
	}

	log.Println(strUuid)
	response.Success(ctx, gin.H {
		"uuid" : strUuid,
		"cnt"  : len(fhs),
	},"上传成功")
}

// 获取图片静态链接
//func GoodsPictureDownload (ctx *gin.Context) {
//	req := struct{
//		uuid string
//		cnt int
//	}{}
//	if err := ctx.Bind(&req); err != nil {
//		log.Println("pictureDownload err: ", err)
//		response.Response(ctx, 422, 422, nil, "参数错误")
//		return
//	}
//
//	if req.uuid == "" || req.cnt == 0 {
//		log.Println("pictureDownload err")
//		response.Response(ctx, 422, 422, nil, "参数错误")
//		return
//	}
//
//	downloadDir := PICTUREDIR + req.uuid
//
//
//}


// 发布商品
func Publish (ctx *gin.Context) {
	var goods model.Goods
	if err := ctx.Bind(&goods); err != nil {
		log.Println("Publish参数错误, ", goods)
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	log.Println("publish params，", goods)

	// 验证token并获取发布者的Email
	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足"})
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
	goods.OwnerEmail = claims.UserEmail

	// 参数校验失败
	if judgeGoodsParameter(ctx, &goods) == false {
		log.Println("PublishGoods参数校验失败")
		return
	}

	response.Success(ctx, gin.H{},"发布成功")

	// 异步写入DB
	go func (localGoods model.Goods) {
		model.GoodsChannel <- struct {} {} // 并发控制
		defer func () {
			<- model.GoodsChannel
		} ()
		if localGoods.Type == 1 { // 如果是丢失物品，价格自动设置为0
			localGoods.Price = 0
		}

		db := common.GetDB()
		c := common.GetRedis()
		defer c.Close()

		newGoods := model.Goods{
			Name: localGoods.Name,
			Price: localGoods.Price,
			WordDetail: localGoods.WordDetail,
			MainPicture: localGoods.MainPicture,
			MorePictures: localGoods.MorePictures,
			OwnerEmail: localGoods.OwnerEmail,
			Deleted: "0",
			Type: localGoods.Type,
		}
		err := db.Create(&newGoods).Error
		if err != nil || newGoods.ID == 0 {
			log.Println("插入NewGoods DB失败, ERR = ",err) // todo xutianmeng 如果写入MYSQL失败是不是应该算作插入失败 ，需要重新插入
			return
		}
		script := `
			local key1 = "GoodsIDList" .. KEYS[1] 
			redis.call('LPUSH', key1, ARGV[1]) 
			local key = "GoodsID" .. ARGV[1]
			redis.call('HSET', key, 'name', ARGV[2]) 
			redis.call('HSET', key, 'price', ARGV[3]) 
			redis.call('HSET', key, 'wordDetail', ARGV[4])  
			redis.call('HSET', key, 'mainPicture', ARGV[5]) 
			redis.call('HSET', key, 'morePicture', ARGV[6]) 
			redis.call('HSET', key, 'type', ARGV[7]) 
			redis.call('HSET', key, 'email', KEYS[1])
			if ARGV[7] == "1" then 
				redis.call('ZADD', "lostZset", ARGV[1] * -1, ARGV[1])
			elseif ARGV[7] == "2" then 
				redis.call('ZADD', "electronicZset", ARGV[1] * -1, ARGV[1]) 
			elseif ARGV[7] == "3" then 
				redis.call('ZADD', "liveZset", ARGV[1] * -1, ARGV[1]) 
			elseif ARGV[7] == "4" then 
				redis.call('ZADD', "foodZset", ARGV[1] * -1, ARGV[1]) 
			end `
		_, err = c.Do("EVAL", script, "1", localGoods.OwnerEmail, newGoods.ID, newGoods.Name, newGoods.Price, newGoods.WordDetail, newGoods.MainPicture, newGoods.MorePictures, newGoods.Type)
		if err != nil {
			log.Println("publishGoods插入redis失败,ERR = ", err)
		}
	} (goods)
}

// 根据分类展示商品 主页需要的数据 不校验权限
// 获取指定的分类1234，pageNumber和Rn 第几页和一页有多少个
func GoodsDisplay (ctx *gin.Context) {
	var requestGoods = GoodsRequest{}

	err := ctx.Bind(&requestGoods)
	if err != nil {
		log.Println(err)
	}
	log.Println("requestGoods: ",requestGoods)
	requestKind := requestGoods.GoodsType
	requestPn   := requestGoods.Pn
	requestRn   := requestGoods.Rn

	// 校验参数
	if requestKind == "" || requestPn == "" || requestRn == "" {
		//log.Println("GoodsDisplay: ",requestKind, " ", requestPn , " " , requestRn)
		response.Response(ctx, 422, 422, nil, "参数缺失")
		return
	}
	if len(requestRn) > 2 || len(requestPn) > 2 || len(requestKind) > 1 || (requestKind != "1" && requestKind != "2" &&
		requestKind != "3" && requestKind != "4") {
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	//参数校验 防止转换时出现错误 todo 不要嫌麻烦在这种不起眼但很关键的判断上
	kind, err := strconv.Atoi(requestKind)
	if err != nil {
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}
	pn, err := strconv.Atoi(requestPn)
	if err != nil {
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}
	rn, err := strconv.Atoi(requestRn)
	if err != nil {
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	// redis ZRANGE KEY START STOP
	c := common.GetRedis()
	defer c.Close()

	// ZREVRANGE lostZset lostZset (pn-1)*rn+1 pn*rn
	// 从大到小排序 通过zset实现分页获取
	start := (pn-1) * rn
	stop := pn * rn - 1

	ID  	 := make([]int, 0)
	email 	 := make([]string, 0)
	names    := make([]string, 0)
	prices   := make([]float64, 0)
	pictures := make([]string, 0)


	script := `
		local type = KEYS[1]
		local start = KEYS[2]
		local stop = KEYS[3]
		local IDs = {}
		if type == "1" then
			IDs = redis.call('ZREVRANGE', 'lostZset', start, stop)
		elseif type == "2" then
			IDs = redis.call('ZREVRANGE', 'electronicZset', start, stop)
		elseif type == "3" then
			IDs = redis.call('ZREVRANGE', 'liveZset', start, stop)
		elseif type == "4" then
			IDs = redis.call('ZREVRANGE', 'foodZset', start, stop)
		end
		local res = {}
		for i = 1, #IDs do
			local ID = IDs[i]
			local name = redis.call('HGET', "GoodsID" .. ID, "name")
			local price = redis.call('HGET', "GoodsID" .. ID, "price")
			local mainPicture = redis.call('HGET', "GoodsID" .. ID, "mainPicture")
			local email = redis.call('HGET', 'GoodsID' .. ID, "email")
			res[#res+1] = name .. "," .. price .. "," .. mainPicture .. "," .. ID .. "," .. email
		end
		return res`
	res, err := redis.Values(c.Do("EVAL", script, "3", kind, start, stop))

	if err != nil { // 如果redis查询失败，重新去mysql查询。
		log.Println("查询失败, err = ", err)
		// mysql select name, price from goods where type = ? and deleted = 0 limit start, stop
		db := common.GetDB()
		var tempGoods []model.Goods
		db.Select("ID, owner_email, name, price, main_picture").Where("type = ? and deleted = 0", kind).Find(&tempGoods).Offset((pn-1)*rn).Limit(rn)
		for _, v := range tempGoods {
			names    = append(names, v.Name)
			prices   = append(prices, v.Price)
			pictures = append(pictures, v.MainPicture)
			ID       = append(ID, int(v.ID))
			email    = append(email, v.OwnerEmail)
		}
		response.Success(ctx, gin.H{
			"cnt" 	  : len(names),
			"name"    : names,
			"price"   : prices,
			"picture" : pictures,
			"ID"	  : ID,
			"email"   : email,
		}, "查询成功")
		return
	}

	// 从redis lua脚本中拿到的数据是一个字符串数组，每一个字符串都是「商品名称,商品价格」用逗号分隔
	for _, v := range res {
		if v == nil {
			continue
		}
		str      := string(v.([]byte))
		temp     := strings.Split(str, ",")
		names     = append(names, temp[0])
		price, _ := strconv.ParseFloat(temp[1], 64) // redis读取的数据都是字符串，修改为float。 mysql中price的数据类型是float
		prices    = append(prices, price)
		pictures  = append(pictures, temp[2])
		tmp, _   := strconv.Atoi(temp[3])
		ID		  = append(ID, tmp)
		email     = append(email, temp[4])
	}
	response.Success(ctx, gin.H{
		"cnt" : len(names),
		"name" : names,
		"price": prices,
		"picture" : pictures,
		"ID"	  : ID,
		"email"   : email,
	}, "查询成功")
}

// 根据userEmail获取该用户所有发布商品
// 查询本人信息传入的字符串是myself，通过获取token拿到UserEmail
func GoodsInfo (ctx *gin.Context) {
	requestEmail := ctx.Param("email")
	if requestEmail == "" {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "邮箱参数缺失")
		return
	}
	if len(requestEmail) > 30 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "邮箱参数错误")
		return
	}
	if requestEmail == "myself" { // 查询本人发布商品信息，传入的是myself,但是需要有token
		tokenString := ctx.GetHeader("Authorization")
		if tokenString == "" || len(tokenString) <= 7 {
			response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "没有权限")
			return
		}
		if strings.HasPrefix(tokenString,"Bearer ") {
			tokenString = tokenString[7:]
		}
		log.Println("info Token = ", tokenString)
		token, claims, err := common.ParseToken(tokenString)
		if err != nil || !token.Valid {
			ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
			return
		}
		requestEmail = claims.UserEmail
	}

	log.Println("info token = ", ctx.GetHeader("Authorization"))

	db := common.GetDB()
	c := common.GetRedis()
	defer c.Close() // ?

	if !isExist(db, requestEmail) {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "邮箱不存在 " + requestEmail)
		return
	}

	// 数据校验通过
	script := `local ids = redis.call('LRANGE', "GoodsIDList" .. KEYS[1], 0, -1)
		local res = {}
		for i = 1, #ids do
			local idKey = "GoodsID" .. ids[i]
			local name = redis.call('HGET', idKey, 'name')
			local price = redis.call('HGET', idKey, 'price')
			local wordDetail = redis.call('HGET', idKey, 'wordDetail')
			local mainPicture = redis.call('HGET', idKey, 'mainPicture')
			local morePicture = redis.call('HGET', idKey, 'morePicture')
			local type = redis.call('HGET', idKey, 'type')
			res[#res+1] = name..","..ids[i]..","..price..","..wordDetail..","..mainPicture..","..morePicture..","..type
		end
		return res`

	var ids []int
	var names []string
	var prices []float64
	var wordDetails []string
	var mainPicture []string
	var morePicture []string
	var kinds []int

	res, err := redis.Values(c.Do("EVAL", script, "1", requestEmail))
	if err != nil {
		log.Println("redis GoodsInfo查询失败，err = ", err)
		// find from mysql
		// select name, price, word_detail, main_picture, more_picture, type from goods where deleted = 0 and owner_email = ?
		var goods []model.Goods
		db.Select("name, price, word_detail, main_picture, more_pictures, type").
			Where("deleted = 0 and owner_email = ?", requestEmail).Find(&goods)
		for _, v := range goods {
			names = append(names, v.Name)
			ids = append(ids, int(v.ID))
			prices = append(prices, v.Price)
			wordDetails = append(wordDetails, v.WordDetail)
			mainPicture = append(mainPicture, v.MainPicture)
			morePicture = append(morePicture, v.MorePictures)
			kinds = append(kinds, v.Type)
		}
		response.Success(ctx, gin.H {
			"ids" : ids,
			"names" : names,
			"prices": prices,
			"wordDetails" : wordDetails,
			"mainPictures" : mainPicture,
			"morePictures" : morePicture,
			"types" : kinds,
		}, "查询成功")
		return
	}

	for _, v := range res {
		if v == nil { // 如果为nil直接返回
			continue
		}
		temp := strings.Split(string(v.([]byte)), ",")
		names = append(names, temp[0])

		tt, _ := strconv.Atoi(temp[1])
		ids = append(ids, tt)

		price, _ := strconv.ParseFloat(temp[2], 64)
		prices = append(prices, price)

		wordDetails = append(wordDetails, temp[3])

		mainPicture = append(mainPicture, temp[4])

		morePicture = append(morePicture, temp[5])

		kind, _ := strconv.ParseInt(temp[6], 10, 0)
		kinds = append(kinds, int(kind))
	}
	response.Success(ctx, gin.H{
		"ids" : ids,
		"name" : names,
		"price": prices,
		"wordDetail" : wordDetails,
		"mainPicture" : mainPicture,
		"morePicture" : morePicture,
		"type" : kinds,
	}, "查询成功")
}

// 修改指定商品的信息，前端必须传入ID主键
// 删除商品就传入DELETED = 1
// 对数据的修改操作还是要加锁！！！ 读的情况不保证读到最新数据，用户刷新一下就OK，不可以出现脏写
func GoodsAlterInfo (ctx *gin.Context) {
	tokenString := ctx.GetHeader("Authorization")

	var requestGoods model.Goods
	if err := ctx.Bind(&requestGoods); err != nil {
		log.Println("GoodsAlterInfo参数错误")
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	log.Println("GoodsAlterInfo: ", requestGoods)
	log.Println("goods.GoodsID = ", requestGoods.GoodsID)
	log.Println("goods.MorePictures = ", requestGoods.MorePictures)
	requestGoods.ID = uint(requestGoods.GoodsID)

	if tokenString == "" {
		response.Response(ctx, 422, 422, nil, "参数缺失")
		return
	}

	// 验证权限
	if strings.HasPrefix(tokenString,"Bearer ") {
		tokenString = tokenString[7:]
	}
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo": "err = " + err.Error()})
		return
	}

	requestGoods.OwnerEmail = claims.UserEmail

	// 获取redis锁
	c := common.GetRedis()
	defer c.Close()

	// 没获取到锁就退出
	script := `
		local t = redis.call('EXISTS', KEYS[1])
		if t == 0 then
		    redis.call('SET', KEYS[1], 1)
		    redis.call('PEXPIRE', KEYS[1], 200) -- 过期时间的设定需要考虑实际业务速度
		    return 1
		end 
		return 0`
	if flag, err := redis.Int(c.Do("EVAL", script, "1", "goodsLock" + strconv.Itoa(int(requestGoods.GoodsID)))) ; err != nil || flag != 1  {
		log.Println("获取分布式锁失败, err = ", err)
		response.Response(ctx, 503, 503, nil, "服务器错误")
		return
	}
	if requestGoods.Deleted ==  "1" {
		response.Success(ctx,  gin.H{}, "删除信息成功")
		go func (localGoods model.Goods) {
			model.GoodsChannel <- struct {} {}
			defer func () {
				<- model.GoodsChannel
			} ()
			// mark deleted for mysql
			db := common.GetDB()
			c  := common.GetRedis()
			defer c.Close()
			alterField := make(map[string]interface{})
			alterField["deleted"] = 1
			//db.Model(&model.Goods{}).Where("id = ?", uint(localGoods.GoodsID)).Updates(alterField)
			db.Where("id = ?", uint(localGoods.GoodsID)).Delete(&model.Goods{})
			log.Println(localGoods.GoodsID)
			// redis
			script := `
				-- 删除商品信息的缓存
				-- KEYS[1] = ID KEYS[2] = email KEYS[3] = type
				redis.call('LREM', "GoodsIDList"..KEYS[2], "0", KEYS[1]) -- 删掉用户商品ID队列中的ID
				redis.call('DEL', "GoodsID"..KEYS[1]) -- 删掉商品详细信息
				local zsetKey = ""
				if KEYS[3] == "1" then
					zsetKey = "lostZset"
				elseif KEYS[3] == "2" then
					zsetKey = "electronicZset"
				elseif KEYS[3] == "3" then
					zsetKey = "liveZset"
				elseif KEYS[3] == "4" then
					zsetKey = "foodZset"
				end
				redis.call('ZREM', zsetKey, KEYS[1]) -- 删除zset中的ID` // TODO xutianmeng 这里缺少冒号 不确定有没有问题
			_, err := c.Do("EVAL", script, "3", localGoods.GoodsID, localGoods.OwnerEmail, localGoods.Type)
			if err != nil {
				log.Println("删除缓存失败， err = ", err.Error()) // TODO xutianmeng list
			}
			script = "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
			_, err = c.Do("EVAL", script, "1", "goodsLock" + strconv.Itoa(int(requestGoods.GoodsID)))
			if err != nil {
				log.Println("释放分布式锁失败, GoodsID = ", requestGoods.GoodsID, ", err = ", err.Error())
			}
		} (requestGoods)
		return
	}
	// 校验通过
	response.Success(ctx,  gin.H{}, "修改信息成功")

	go func (localGoods model.Goods) {
		model.GoodsChannel <- struct {} {}
		defer func () {
			<- model.GoodsChannel
		} ()
		db := common.GetDB()
		c := common.GetRedis()
		defer c.Close()
		alterFields := make(map[string]interface{})
		if localGoods.Name != "" {
			alterFields["name"] = localGoods.Name
		}
		if localGoods.Price != 0 {
			alterFields["price"] = localGoods.Price
		}
		if localGoods.Type != 0 {
			alterFields["type"] = localGoods.Type
		}
		if localGoods.MainPicture != "" {
			alterFields["main_picture"] = localGoods.MainPicture
		}
		if localGoods.MorePictures != "" {
			alterFields["more_pictures"] = localGoods.MorePictures
		}
		if localGoods.WordDetail != "" {
			alterFields["word_detail"] = localGoods.WordDetail
		}
		db.Model(&model.Goods{}).Where("deleted = 0 and id = ?", localGoods.GoodsID).Updates(alterFields)
		// redis
		script := `
			-- 更新goods信息到redis
			if ARGV[1] ~= "" then
				redis.call('HSET', KEYS[1], "name", ARGV[1])
			end
			if ARGV[2] ~= "" then
				redis.call('HSET', KEYS[1], "price", ARGV[2])
			end
			if ARGV[3] ~= "" then
				redis.call('HSET', KEYS[1], "type", ARGV[3])
			end
			if ARGV[4] ~= "" then
				redis.call('HSET', KEYS[1], "mainPicture", ARGV[4])
			end
			if ARGV[5] ~= "" then
				redis.call('HSET', KEYS[1], "morePicture", ARGV[5])
			end
			if ARGV[6] ~= "" then
				redis.call('HSET', KEYS[1], "wordDetail", ARGV[6])
			end`
		_, err := c.Do("EVAL", script, "1", "GoodsID" + strconv.Itoa(int(localGoods.GoodsID)), localGoods.Name,
			localGoods.Price, localGoods.Type, localGoods.MainPicture, localGoods.MorePictures,
			localGoods.WordDetail)
		if err != nil {
			log.Println("redis更新数据失败")
		}
		script = "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
		_, err = c.Do("EVAL", script, "1", "goodsLock" + strconv.Itoa(int(requestGoods.GoodsID)))
		if err != nil {
			log.Println("释放分布式锁失败, GoodsID = ", requestGoods.GoodsID, ", err = ", err.Error())
		}
	} (requestGoods)
}

// 商品详情页
func GoodsInfoDetail (ctx *gin.Context) {
	var requestGoods = GoodsRequest{}

	if err := ctx.Bind(&requestGoods); err != nil {
		log.Println("GoodsDetail参数错误 ", requestGoods) // debug
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}
	log.Println("GoodsInfoDetail: ", requestGoods)
	requestID	 := requestGoods.Id
	requestEmail := requestGoods.Email

	if requestEmail == "" || len(requestEmail) > 30 || requestID == "" {
		response.Response(ctx, 422, 422, nil, "参数缺失")
		return
	}
	c := common.GetRedis()
	defer c.Close()
	//script := "local name = redis.call('HGET', 'GoodsID' .. KEYS[1], \"name\")\nlocal price = redis.call('HGET', 'GoodsID' .. KEYS[1], \"price\")\nlocal type = redis.call('HGET', 'GoodsID' .. KEYS[1], \"type\")\nlocal mainPicture = redis.call('HGET', 'GoodsID' .. KEYS[1], \"mainPicture\")\nlocal morePicture = redis.call('HGET', 'GoodsID' .. KEYS[1], \"morePicture\")\nlocal wordDetail = redis.call('HGET', 'GoodsID' .. KEYS[1], \"wordDetail\")\nlocal email = redis.call('HGET', 'GoodsID' .. KEYS[1], \"email\")\nreturn name .. \",\" .. price .. \",\" .. type .. \",\" .. mainPicture .. \",\" .. morePicture .. \",\" .. wordDetail .. \",\" .. email"
	script := `
		local name = redis.call('HGET', 'GoodsID' .. KEYS[1], "name")
		local price = redis.call('HGET', 'GoodsID' .. KEYS[1], "price")
		local type = redis.call('HGET', 'GoodsID' .. KEYS[1], "type")
		local mainPicture = redis.call('HGET', 'GoodsID' .. KEYS[1], "mainPicture")
		local morePicture = redis.call('HGET', 'GoodsID' .. KEYS[1], "morePicture")
		local wordDetail = redis.call('HGET', 'GoodsID' .. KEYS[1], "wordDetail")
		local email = redis.call('HGET', 'GoodsID' .. KEYS[1], "email")
		return name .. "," .. price .. "," .. type .. "," .. mainPicture .. "," .. morePicture .. "," .. wordDetail .. "," .. email
`
	res, err := redis.String(c.Do("EVAL", script, "1", requestID))
	if err != nil {
		log.Println("redis GoodsInfoDetail failed, err = ", err) // find from mysql
		db := common.GetDB()
		var tempGoods model.Goods
		db.Select("name, price, main_picture, more_pictures, type, word_detail, owner_email").
			Where("id = ? and deleted = 0", requestID).First(&tempGoods)
		response.Success(ctx, gin.H{
			"name" : tempGoods.Name,
			"price" : tempGoods.Price,
			"type" : tempGoods.Type,
			"mainPicture" : tempGoods.MainPicture,
			"morePictures" : tempGoods.MorePictures,
			"wordDetail" : tempGoods.WordDetail,
			"ownerEmail" : tempGoods.OwnerEmail,
		}, "查询商品详情成功")
		return
	}
	temp := strings.Split(res, ",")
	name := temp[0]
	price, _ := strconv.ParseFloat(temp[1], 64)
	kind, _ := strconv.Atoi(temp[2])
	mainPicture := temp[3]
	morePictures := temp[4]
	wordDetail := temp[5]
	email := temp[6]

	response.Success(ctx, gin.H{
		"name" : name,
		"price" : price,
		"type" : kind,
		"mainPicture" : mainPicture,
		"morePictures" : morePictures,
		"wordDetail" : wordDetail,
		"ownerEmail" : email,
	}, "查询商品详情成功")
}

// 获取ID和token判断是否是本人发的商品
func GoodsAuth (ctx *gin.Context) {
	var requestGoods = GoodsRequest{}

	if err := ctx.Bind(&requestGoods); err != nil {
		log.Println("GoodsAuth参数错误") // debug
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	//requestID	 := requestGoods.Id
	requestEmail := requestGoods.Email
	tokenString  := ctx.GetHeader("Authorization")
	if requestEmail == "" || len(tokenString) < 7 {
		response.Response(ctx, 422, 422, nil, "参数缺失")
		return
	}
	tokenString =  tokenString[7:]
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid || claims.UserEmail != requestEmail {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
		return
	}

	response.Success(ctx, gin.H{
		"hasAuth" : true,
	}, "该用户有此商品的权限进行修改")
}

// 获取每一种商品总共有多少个
func GoodsTypeCount (redisKey string) int {
	c := common.GetRedis()
	defer func () {
		if err := c.Close(); err != nil {
			log.Println("GoodsTypeCount close redisConnection failed, err  = ", err)
		}
	} ()
	ret, err := redis.Int(c.Do("ZCARD", redisKey))
	if err != nil {
		log.Println(err)
	}
	return ret
}

// 分页获取数据 用于推荐商品展示
func RecommendGoods (redisKey string, pn, rn int) []string {
	c := common.GetRedis()
	defer func () {
		if err := c.Close(); err != nil {
			log.Println("GoodsTypeCount close redisConnection failed, err  = ", err)
		}
	} ()
	script := "local type = KEYS[1] " +
		"local start = KEYS[2] " +
		"local stop = KEYS[3] " +
		"local IDs = {} " +
		"IDs = redis.call('ZREVRANGE', type, start, stop) " +
		"local res = {} " +
		"for i = 1, #IDs do " +
		"    local ID = IDs[i] " +
		"    local name = redis.call('HGET', \"GoodsID\" .. ID, \"name\") " +
		"    local price = redis.call('HGET', \"GoodsID\" .. ID, \"price\") " +
		"    local mainPicture = redis.call('HGET', \"GoodsID\" .. ID, \"mainPicture\") " +
		"    res[#res+1] = name .. \",\" .. price .. \",\" .. mainPicture " +
		"end " +
		"return res"
	res, err := redis.Values(c.Do("EVAL", script, redisKey, pn * rn, (pn+1)*rn-1))
	if err != nil {
		log.Println(err)
	}
	var ret []string
	for _, v := range res {
		if v != nil {
			ret = append(ret, string(v.([]byte)))
		}
	}
	return ret
}

func judgeGoodsParameter (ctx *gin.Context, localGoods *model.Goods) bool {
	if localGoods.OwnerEmail == "" {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "发布者邮箱不能为空")
		return false
	}
	if len(localGoods.OwnerEmail) > 50 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "发布者邮箱不能超过50个字符")
		return false
	}
	if localGoods.Name == "" {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "商品名称不能为空")
		return false
	}
	if len(localGoods.Name) >= 99 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "商品名称字数不能超过100个字符")
		return false
	}
	if localGoods.MainPicture == "" {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "主图片不能为空")
		return false
	}
	if localGoods.Price < 0 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "价格不能为负数")
		return false
	}
	if localGoods.Price >= 100000 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "价格过高，请重设置")
		return false
	}
	if localGoods.WordDetail == "" {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "文字简介不能为空")
		return false
	}
	if len(localGoods.WordDetail) >= 499 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "文字简介不能超过500个字符")
		return false
	}
	if localGoods.Type < 0 || localGoods.Type > 3 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "商品分类设置错误")
		return false
	}
	return true
}

