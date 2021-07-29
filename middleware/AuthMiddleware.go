package middleware

import (
	"TradePlatform/common"
	"TradePlatform/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 获取authorization header
		tokenString := ctx.GetHeader("Authorization")

		// validate token formate
		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足"})
			ctx.Abort()
			return
		}

		tokenString = tokenString[7:]

		token, claims, err := common.ParseToken(tokenString)
		if err != nil || !token.Valid {
			ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足"})
			ctx.Abort()
			return
		}

		c := common.GetRedis()
		defer c.Close()
		//script := "local ans = {}\nans[#ans+1] = redis.call('HGET', KEYS[1], \"name\")\nans[#ans+1] = redis.call('HGET', KEYS[1], 'telephone')\nans[#ans+1] = redis.call('HGET', KEYS[1], 'sid')\nans[#ans+1] = redis.call('HGET', KEYS[1], 'password')\nans[#ans+1] = redis.call('HGET', KEYS[1], 'class')\nans[#ans+1] = redis.call('HGET', KEYS[1], 'signature')\nreturn ans"


		// 验证通过后获取claim 中的userId
		userId := claims.UserId
		DB := common.GetDB()
		var user model.User
		DB.First(&user, userId)

		// 用户
		if user.ID == 0 {
			ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足"})
			ctx.Abort()
			return
		}

		// 用户存在 将user 的信息写入上下文
		ctx.Set("user", user)

		ctx.Next()
	}
}
