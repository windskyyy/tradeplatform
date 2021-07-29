package main

import (
	"TradePlatform/controller"
	"TradePlatform/middleware"
	"github.com/gin-gonic/gin"
)

func CollectRoute(router *gin.Engine) *gin.Engine {
	router.Use(middleware.OnceRequest()) // 中间件

	userInfo := router.Group("/api/auth")
	{
		userInfo.GET("/token/:mytoken", controller.InfoByToken) // 根据 token获取用户信息
		userInfo.POST("/login", middleware.BloomFilter(), controller.Login) // 登录
		userInfo.POST("/register", controller.Register) // 注册
		userInfo.GET("/info/:email", middleware.BloomFilter(), controller.Info) // 查询个人信息
		userInfo.POST("/alterinfo", middleware.BloomFilter(), controller.AlterInfo) 	// 修改个人信息
		userInfo.POST("/alterpassword", middleware.BloomFilter(), controller.AlterPassword) // 修改密码
		userInfo.POST("/recommend", controller.UserRecommend) // 布隆过滤器实现推荐算法
	}

	goodsInfo := router.Group("/api/goods")
	{
		goodsInfo.POST("/publish", controller.Publish) // 发布商品
		goodsInfo.GET("/info/:email", middleware.BloomFilter(), controller.GoodsInfo) // 查看某用户发布所有商品
		goodsInfo.POST("/alter", middleware.BloomFilter(), controller.GoodsAlterInfo) // 修改商品信息
		goodsInfo.POST("/display", controller.GoodsDisplay) // 首页展示商品，分页展示
		goodsInfo.POST("/detail", middleware.BloomFilter(), controller.GoodsInfoDetail) // 查看商品详情
		goodsInfo.POST("/auth", middleware.BloomFilter(), controller.GoodsAuth) // 判断商品发布者与当前用户关系
		goodsInfo.POST("/uploads", controller.GoodsPictureUploads) //上传多张图片
		goodsInfo.OPTIONS("/uploads")
	}

	cartInfo := router.Group("/api/cart")
	{
		cartInfo.POST("/add", middleware.BloomFilter(), controller.AddCart) // 添加购物车
		cartInfo.POST("/del", middleware.BloomFilter(), controller.DelCart) // 删除购物车中的商品
		cartInfo.POST("/info", middleware.BloomFilter(), controller.CartInfo) // 查看购物车信息

	}

	return router
}

