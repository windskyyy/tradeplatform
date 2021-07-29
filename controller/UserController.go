package controller

import (
	"TradePlatform/common"
	"TradePlatform/model"
	"TradePlatform/response"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
	"log"
	"net/http"
	"strings"
)

// 输入token 返回用户信息
func InfoByToken (ctx *gin.Context) {
	tokenString := ctx.Param("mytoken")

	if tokenString == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "token为空"})
		log.Println("token为空")
		return
	}
	if strings.HasPrefix(tokenString,"Bearer ") {
		tokenString = tokenString[7:]
	}
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid {
		log.Println("token not Valid")
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
		return
	}
	response.Success(ctx, gin.H {
		"UserEmail": claims.UserEmail,
	}, "查询成功")
}

// 注册时先获取分布式锁，然后校验权限
// 传入所有参数
func Register (ctx *gin.Context) {
	var requestUser = model.User{}
	if err := ctx.Bind(&requestUser); err != nil {
		log.Println("Register参数错误. requestUser = ", requestUser) // debug
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}
	log.Println("userRegister : ", requestUser) // debug

	if requestUser.UserEmail == "" {
		log.Println("UserEmail为空, request = ", requestUser) // debug
		response.Response(ctx, http.StatusInternalServerError, 400, nil, "参数错误")
		return
	}

	// 获取锁
	c := common.GetRedis()
	script := `
		local t = redis.call('EXISTS', KEYS[1])
			if t == 0 then 
				redis.call('SET', KEYS[1], 1)
				redis.call('PEXPIRE', KEYS[1], 100) -- 过期时间的设定需要考虑实际业务速度 
			return 1 
		end
		return 0`
	if getLock, err := redis.Int(c.Do("EVAL", script, "1", "lock" + requestUser.UserEmail)); err != nil || getLock == 0 {
		if err != nil {
			log.Println(requestUser)
			log.Println("获取分布式锁失败, err = ", err)
		}
		response.Response(ctx, http.StatusInternalServerError, 500, nil, "系统异常")
		c.Close()
		return
	}

	// 校验数据, 失败则释放锁
	if judgeParameter(&requestUser, ctx) == false {
		script := "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
		_, err := c.Do("EVAL", script, "1", "lock" + requestUser.UserEmail)
		if err != nil {
			log.Println("释放分布式锁失败, Email = ", requestUser.UserEmail, ", err = ", err)
		}
		response.Response(ctx, 422, 422, nil, "参数错误")
		c.Close()
		return
	}
	db := common.GetDB()
	// 判断邮箱是否存在
	if isExist(db, requestUser.UserEmail) {
		_, err := c.Do("EVAL", script, "1", "lock" + requestUser.UserEmail)
		if err != nil {
			log.Println("释放分布式锁失败, Email = ", requestUser.UserEmail, ", err = ", err)
		}
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "用户已经存在")
		c.Close()
		return
	}

	c.Close()

	// 发放token
	token, err := common.ReleaseToken(requestUser)
	if err != nil {
		log.Printf("token generate error : %v", err)
		response.Response(ctx, http.StatusInternalServerError, 500, nil, "系统异常")
		return
	}

	// 校验数据通过,异步返回结果
	response.Success(ctx,  gin.H{"token": token}, "注册成功")

	// 异步写入db
	go func (db *gorm.DB, localRequestUser model.User) {
		// 限制并发量
		model.UserChannel <- struct {}{}
		common.GlobalBloomFilter.AddString(localRequestUser.UserEmail)
		newUser := model.User{
			UserName: localRequestUser.UserName,
			UserEmail: localRequestUser.UserEmail,
			UserTelephone: localRequestUser.UserTelephone,
			UserSid: localRequestUser.UserSid,
			UserPassword: localRequestUser.UserPassword,
			UserClass: localRequestUser.UserClass,
			UserSignature: localRequestUser.UserSignature,
			Deleted: localRequestUser.Deleted,
		}
		err := db.Create(&newUser).Error // 写入mysql
		if err != nil {
			log.Println("User Register Failed. err = ", err)
		}
		// 缓存到redis并释放掉分布式锁
		redisScript := "redis.call('HSET', KEYS[1], \"name\", ARGV[1]);\nredis.call('HSET', KEYS[1], \"telephone\", ARGV[2]);\nredis.call('HSET', KEYS[1], \"sid\", ARGV[3]);\nredis.call('HSET', KEYS[1], \"password\", ARGV[4]);\nredis.call('HSET', KEYS[1], \"class\", ARGV[5]);\nredis.call('HSET', KEYS[1], 'signature', ARGV[6])\nif redis.call('EXISTS', KEYS[2]) == 1 then\n    redis.call('DEL', KEYS[2])\nend"
		c := common.GetRedis()
		defer func () {
			err := c.Close()
			if err != nil {
				log.Println("关闭redis失败, err = ", err.Error())
			}
		} ()
		_, err = c.Do("EVAL", redisScript, "2", localRequestUser.UserEmail, "lock" + localRequestUser.UserEmail, localRequestUser.UserName, localRequestUser.UserTelephone, localRequestUser.UserSid, localRequestUser.UserPassword, localRequestUser.UserClass, localRequestUser.UserSignature)
		if err != nil {
			log.Println("插入redis用户注册信息失败")
		}
		<- model.UserChannel
	} (db, requestUser)
}

// 传入邮箱密码
func Login (ctx *gin.Context) {
	var requestUser = model.User{}

	if err := ctx.Bind(&requestUser); err != nil {
		//log.Println("Login参数错误") // debug
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	log.Println("UserLogin: " ,requestUser)

	db := common.GetDB()
	if !isExist(db, requestUser.UserEmail) {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "用户不存在")
		return
	}
	if err := judgePassword(db, requestUser.UserEmail, requestUser.UserPassword); err != nil {  //  密码不正确 or 数据没找到密码
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "密码错误")
		return
	}

	// 发放token
	token, err := common.ReleaseToken(requestUser)
	if err != nil {
		response.Response(ctx, http.StatusInternalServerError, 500, nil, "系统异常")
		log.Printf("token generate error : %v", err)
		return
	}

	// 返回结果
	response.Success(ctx,  gin.H{"token": token}, "登录成功")
}

// 如果是查询本人信息，返回所有信息，可用来提供修改密码服务。非本人不返回密码信息，其他都返回。
// 传入要查询用户的邮箱，通过get参数，通过token来判断是否是查询本人信息。
func Info (ctx *gin.Context) {
	requestEmail := ctx.Param("email") // 被查询信息人的邮箱
	if requestEmail == "" {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "参数缺失")
		return
	}

	log.Println("查询信息，UserEmail = ", requestEmail) // debug

	isMyself := true
	var tempUser model.User
	tokenString := ctx.GetHeader("Authorization")
	tokenString = strings.TrimSpace(tokenString)
	log.Println("UserInfo token=", tokenString) // debug
	if len(tokenString) <= 7 {
		isMyself = false
	} else {
		if strings.HasPrefix(tokenString,"Bearer ") {
			tokenString = tokenString[7:]
		}
		token, claims, err := common.ParseToken(tokenString)
		if err != nil || !token.Valid || claims.UserEmail != requestEmail {
			isMyself = false
		}
	}

	c := common.GetRedis()
	defer c.Close()

	// 判断是否存在该用户邮箱
	ret, err := redis.Int(c.Do("EXISTS", requestEmail))
	if err != nil || ret == 0 {
		log.Println("判断用户信息是否存在redis失败, email = ", requestEmail, ", err = ", err) // debug
		db := common.GetDB()
		var tmpUser model.User
		db.Where("deleted = 0 and user_email = ?", requestEmail).First(&tmpUser)
		if tmpUser.ID == 0 {
			log.Println("mysql中也没有该用户信息, email = ", requestEmail)
			ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户信息不存在"})
			return
		}
	}

	isRedisVal := true
	redisscript := "local ans = {}\nans[#ans+1] = redis.call('HGET', KEYS[1], \"name\")\nans[#ans+1] = redis.call('HGET', KEYS[1], 'telephone')\nans[#ans+1] = redis.call('HGET', KEYS[1], 'sid')\nans[#ans+1] = redis.call('HGET', KEYS[1], 'password')\nans[#ans+1] = redis.call('HGET', KEYS[1], 'class')\nans[#ans+1] = redis.call('HGET', KEYS[1], 'signature')\nreturn ans\n"
	val, err := redis.Values(c.Do("EVAL", redisscript, "1", requestEmail))
	if err != nil {
		log.Println("查询redis个人信息失败， err = ", err) // 去mysql中查询个人信息
		isRedisVal = false
		db := common.GetDB()
		err := db.Select("user_name, user_telephone, user_sid, user_class, user_signature").Where("deleted = 0 and user_email = ?", requestEmail).First(&tempUser).Error
		if err != nil {
			log.Println("查询mysql个人信息失败，err = ", err) // 查无此人
			ctx.JSON(http.StatusOK, gin.H{"code":200, "msg": "没有此人的信息"})
			return
		}
	}
	if isRedisVal {
		if val[0] != nil {
			tempUser.UserName = string(val[0].([]byte))
		}
		if val[1] != nil {
			tempUser.UserTelephone = string(val[1].([]byte))
		}
		if val[2] != nil {
			tempUser.UserSid = string(val[2].([]byte))
		}
		if val[3] != nil {
			tempUser.UserPassword = string(val[3].([]byte))
		}
		if val[4] != nil {
			tempUser.UserClass = string(val[4].([]byte))
		}
		if val[5] != nil {
			tempUser.UserSignature = string(val[5].([]byte))
		}
	}

	if isMyself == false {
		tempUser.UserPassword = "" 		// 非本人抹掉密码，不返回
	}
	//log.Println("redis 查询 UserInfo成功, UserEmail = ", requestEmail) // debug
	response.Success(ctx, gin.H{
		"permission" : isMyself,
		"userInfo"   : tempUser, // todo xutianmeng 返回一个对象前端是否能够接收？
	}, "查询成功")
}

// 可以修改姓名、电话、学号、班级、个性签名
// 传入修改后的信息和用户的邮箱，邮箱用来判断权限
func AlterInfo (ctx *gin.Context) {
	var updateUser model.User
	if err := ctx.Bind(&updateUser); err != nil {
		log.Println("AlterInfo参数错误")
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	log.Println("ALterUserInfo: ", updateUser)

	if updateUser.UserEmail == "" {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "参数缺失")
		return
	}

	// 获取authorization header
	tokenString := ctx.GetHeader("Authorization")
	tokenString = strings.TrimSpace(tokenString)
	if strings.HasPrefix(tokenString,"Bearer ") {
		tokenString = tokenString[7:]
	}

	log.Println(tokenString)

	if tokenString == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足"})
		return
	}
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid || claims.UserEmail != updateUser.UserEmail {
		if err == nil {
			err = errors.New("userEmail不同，请求UserEmail = " + updateUser.UserEmail + ",被修改的UserEmail="+claims.UserEmail )
			if !token.Valid {
				err = errors.New("token失效")
			}
		}
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
		return
	}

	// 获取锁
	c := common.GetRedis()
	script := "local t = redis.call('EXISTS', KEYS[1])\nif t == 0 then\n    redis.call('SET', KEYS[1], 1)\n    redis.call('PEXPIRE', KEYS[1], 200) -- 过期时间的设定需要考虑实际业务速度\n    return 1\nend\nreturn 0"
	if getLock, err := redis.Int(c.Do("EVAL", script, "1", "lock" + updateUser.UserEmail)); err != nil || getLock == 0 {
		if err != nil {
			log.Println("获取分布式锁失败, err = ", err.Error())
		}
		response.Success(ctx, gin.H{}, "服务器请求失败，请重试")
		c.Close()
		return
	}
	c.Close()

	response.Success(ctx,  gin.H{}, "修改信息成功")

	go func (localUpdateUser model.User, email string) {
		var tempUser model.User
		db := common.GetDB()
		alterFields := make(map[string]interface{})
		if localUpdateUser.UserName != "" {
			alterFields["user_name"] = localUpdateUser.UserName
		}
		if localUpdateUser.UserTelephone != "" {
			alterFields["user_telephone"] = localUpdateUser.UserTelephone
		}
		if localUpdateUser.UserSid != "" {
			alterFields["user_sid"] = localUpdateUser.UserSid
		}
		if localUpdateUser.UserClass != "" {
			alterFields["user_class"] = localUpdateUser.UserClass
		}
		if localUpdateUser.UserSignature != "" {
			alterFields["user_signature"] = localUpdateUser.UserSignature
		}
		// 修改mysql的数据
		db.Model(&tempUser).Where("user_email = ? and deleted = 0", email).Updates(alterFields)
		// 更新到redis
		script := "if ARGV[1] ~= \"\" then\n    redis.call('HSET', KEYS[1], \"name\", ARGV[1])\nend\nif ARGV[2] ~= \"\" then\nredis.call('HSET', KEYS[1], \"telephone\", ARGV[2])\nend\nif ARGV[3] ~= \"\" then\nredis.call('HSET', KEYS[1], \"sid\", ARGV[3])\nend\nif ARGV[4] ~= \"\" then\nredis.call('HSET', KEYS[1], \"class\", ARGV[4])\nend\nif ARGV[5] ~= \"\" then\nredis.call('HSET', KEYS[1], \"signature\", ARGV[5])\nend"
		c := common.GetRedis()
		defer c.Close()
		_, err := c.Do("EVAL", script, "1", email, localUpdateUser.UserName, localUpdateUser.UserTelephone, localUpdateUser.UserSid, localUpdateUser.UserClass, localUpdateUser.UserSignature)
		if err != nil {
			log.Println("修改个人信息数据更新到redis失败, err = ", err)
			_, err := c.Do("DEL", email)
			if err != nil {
				log.Println("更新失败后尝试删除掉脏数据，也失败了， err = ", err)
			}
		}
		script = "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
		_, err = c.Do("EVAL", script, "1", "lock" + localUpdateUser.UserEmail)
		if err != nil {
			log.Println("释放分布式锁失败, Email = ", localUpdateUser.UserEmail, ", err = ", err.Error())
		}
	} (updateUser, claims.UserEmail)
}

// 传入旧密码和新密码和邮箱，邮箱做权限校验
func AlterPassword (ctx *gin.Context) {
	var updateUser model.User
	if err := ctx.Bind(&updateUser); err != nil {
		log.Println("AlterPassword参数错误")
		response.Response(ctx, 422, 422, nil, "参数错误")
		return
	}

	log.Println("AlterPassword: ", updateUser)

	if updateUser.UserEmail == "" || updateUser.UserPassword == "" || updateUser.UserNewPassword == "" {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "参数缺失")
		return
	}

	// 获取authorization header
	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足"})
		return
	}
	if strings.HasPrefix(tokenString,"Bearer ") {
		tokenString = tokenString[7:]
	}
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid || claims.UserEmail != updateUser.UserEmail {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
		return
	}

	// 获取锁
	c := common.GetRedis()
	script := "local t = redis.call('EXISTS', KEYS[1])\nif t == 0 then\n    redis.call('SET', KEYS[1], 1)\n    redis.call('PEXPIRE', KEYS[1], 200) -- 过期时间的设定需要考虑实际业务速度\n    return 1\nend\nreturn 0"
	if getLock, err := redis.Int(c.Do("EVAL", script, "1", "lock" + updateUser.UserEmail)); err != nil || getLock == 0 {
		if err != nil {
			log.Println("获取分布式锁失败, err = ", err.Error())
		}
		response.Success(ctx, gin.H{}, "服务器请求失败，请重试")
		c.Close()
		return
	}

	db := common.GetDB()
	if !isExist(db, updateUser.UserEmail) {
		script := "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
		_, err := c.Do("EVAL", script, "1", "lock" + updateUser.UserEmail)
		if err != nil {
			log.Println("释放分布式锁失败, Email = ", updateUser.UserEmail, ", err = ", err.Error())
		}
		c.Close()
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "用户不存在")
		return
	}

	if err := judgePassword(db, updateUser.UserEmail, updateUser.UserPassword); err != nil {
		script := "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
		_, err := c.Do("EVAL", script, "1", "lock" + updateUser.UserEmail)
		if err != nil {
			log.Println("释放分布式锁失败, Email = ", updateUser.UserEmail, ", err = ", err.Error())
		}
		c.Close()
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "密码不正确")
		return
	}
	c.Close()

	response.Success(ctx,  gin.H{}, "修改密码成功")

	go func (db *gorm.DB, localUser model.User) {
		var tempUser model.User
		var err error
		// 修改mysql数据
		c := common.GetRedis()
		err = db.Model(&tempUser).Where("user_email = ? and deleted = 0",
			localUser.UserEmail).Update("user_password", localUser.UserNewPassword).Error
		if err != nil {
			log.Println("mysql修改密码出问题，邮箱 = ", localUser.UserEmail, ", err = ", err)
		}

		_, err = c.Do("HSET", localUser.UserEmail, "password", localUser.UserNewPassword)
		if err != nil {
			log.Println("redis修改密码出问题，邮箱 = ", localUser.UserEmail, ", err = ", err)
		}
		// 释放锁
		script := "if redis.call('EXISTS', KEYS[1]) == 1 then\n    redis.call('DEL', KEYS[1])\nend"
		_, err = c.Do("EVAL", script, "1", "lock" + updateUser.UserEmail)
		if err != nil {
			log.Println("释放分布式锁失败, Email = ", updateUser.UserEmail, ", err = ", err.Error())
		}
	} (db, updateUser)
}

func UserRecommend (ctx * gin.Context) {
	tokenString := ctx.GetHeader("Authorization")
	if len(tokenString) < 7 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足"})
		log.Println("token empty")
		return
	}

	if strings.HasPrefix(tokenString,"Bearer ") {
		tokenString = tokenString[7:]
	}
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || !token.Valid {
		log.Println("token 失效")
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足", "moreInfo" : "err = " + err.Error()})
		return
	}

	// 遍历四种类型的商品 总共返回五件推荐的商品
	temp := []string{"lostZset", "electronicZset", "liveZset", "foodZset"}
	hasRecommend := 0
	needRecommend := 5
	var names []string
	var prices []string
	var pictures []string
	for _, key := range temp {
		cnt := GoodsTypeCount(key)
		t := cnt / 5 // t = cnt / 5 向上取整 保证能获取所有商品信息
		if cnt % 5 != 0 {
			t++
		}
		for i := 0; i < t && hasRecommend < needRecommend; i++ {
			// 商品名称，商品价格，商品图片
			res := RecommendGoods(key, i, 5)
			for _, str := range res {
				if str != "" && claims.RecommendBloomFilter.TestString(str) == false {
					temp = strings.Split(str, ",")
					if len(temp) == 3 {
						names = append(names, temp[0])
						prices = append(prices, temp[1])
						pictures = append(pictures, temp[2])
					}
					hasRecommend++
					claims.RecommendBloomFilter.AddString(str) // 内部实现了互斥锁保证并发安全
					if hasRecommend >= needRecommend {
						break
					}
				}
			}
		}
		if hasRecommend >= needRecommend {
			break
		}
	}
	response.Success(ctx, gin.H{
		"numbers" : hasRecommend,
		"names" : names,
		"prices" : prices,
		"pictures" : pictures,
	}, "用户推荐信息正确")

}

func judgeParameter(user *model.User, ctx *gin.Context) bool {
	if len(user.UserTelephone) != 11 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "手机号必须为11位")
		return false
	}
	if len(user.UserPassword) < 6 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "密码不能少于6位")
		return false
	}
	if len(user.UserName) == 0 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "名字未填写")
		return false
	}
	if len(user.UserName) >= 90 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "名字长度错误")
		return false
	}
	if len(user.UserSid) == 0 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "学号未填写")
		return false
	}
	if len(user.UserSid) >= 15 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "学号长度错误")
		return false
	}
	if len(user.UserEmail) == 0 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "邮箱未填写")
		return false
	}
	if len(user.UserEmail) > 30 {
		response.Response(ctx, http.StatusUnprocessableEntity, 422, nil, "邮箱长度错误")
		return false
	}
	return true
}

func judgePassword (db *gorm.DB, email, inputpwd string) error {
	c := common.GetRedis()
	defer c.Close()
	pwd := ""
	pwd, err := redis.String(c.Do("HGET", email, "password"))
	if err == nil { // find pwd from redis
		if inputpwd != pwd {
			return errors.New("密码不正确")
		}
		return nil
	}
	log.Println("redis查询密码失败，err = ", err)
	var tempUser model.User
	tempdb := db.Select("user_password").Where("user_email = ? and deleted = 0", "1522972330@qq.com").Find(&tempUser)
	if tempdb.Error == nil && tempUser.ID != 0 {  	// find pwd from mysql
		if inputpwd != tempUser.UserPassword {
			return errors.New("密码不正确")
		}
		return nil
	}
	log.Println("mysql find password failed. err = ", err)
	return errors.New("未找到密码")
}

func isExist(db *gorm.DB, email string) bool {
	c := common.GetRedis()
	defer c.Close()
	flag, err := redis.Int(c.Do("exists", email))
	if err != nil {
		log.Println("查询email是否存在失败， err = ", err.Error())
	}
	if flag == 1 {
		return true
	}
	// todo 不存在就要渗透到mysql？ 这样可能对mysql压力太大了 以后再改把
	var user model.User
	db.Where("user_email = ? and deleted = 0", email).First(&user)
	if user.ID != 0 {
		return true
	}
	return false
}