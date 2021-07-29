package middleware

import (
	"github.com/gin-gonic/gin"
)

func BloomFilter () gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//requestEmail := ctx.Param("email")
		//common.GlobalBloomFilter.AddString("myself")
		//// 使用GET请求 URL带有EMAIL参数 并且EMAIL没有注册过
		//if requestEmail != "" && common.GlobalBloomFilter.TestString(requestEmail) == false {
		//	log.Println("通过GET请求获取EMAIL参数 请求失败，EMAIL不存在")
		//	ctx.Abort()
		//	return
		//}
		//postEmail := ctx.PostForm("email")
		//if postEmail != "" && common.GlobalBloomFilter.TestString(postEmail) == false {
		//	log.Println("通过POST请求获取EMAIL参数，请求失败，EMAIL不存在")
		//	ctx.Abort()
		//	return
		//}
	}
}
