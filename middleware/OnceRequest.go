package middleware

import (
	"github.com/gin-gonic/gin"
)

// 保证同一个IP在X毫秒内只处理一个请求
func OnceRequest () gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//ip := ctx.ClientIP()
		//c := common.GetRedis()
		//defer c.Close()
		//nowTime := time.Now().UnixNano() / 1e6
		//redisScript := "if redis.call('EXISTS', KEYS[1]) == 0 then\n    redis.call('SET', KEYS[1], ARGV[1])\n    return 1\nend\nlocal lasttime = redis.call('GET', KEYS[1])\nif ARGV[1] - lasttime > 300 then\n    redis.call('SET', KEYS[1], ARGV[1])\n    return 1\nend\nreturn 0"
		//flag, err := redis.Int(c.Do("EVAL", redisScript, "1", "IP"+ip, nowTime))
		//if err != nil {
		//	log.Println("幂等性中间件出错，IP = ", ip, ", err = ", err.Error())
		//	flag = 0
		//}
		//if flag == 0 {
		//	//ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "权限不足"})
		//	log.Println("阻止了一次频繁访问， ip = ", ip)
		//	ctx.Abort()
		//	//return
		//}
	}
}