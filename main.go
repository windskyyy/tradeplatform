package main

import (
	"TradePlatform/common"
	"TradePlatform/model"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"os"
)

func main() {
	initConfig()
	common.InitDB()
	// test
	defer func () {
		common.CloseDB()
	} ()

	r := gin.Default()
	r = CollectRoute(r)
	r.Static("/static/upload", "/home/ubuntu/workspace/GoLang/src/TradePlatform/goodsPicture/")


	port := viper.GetString("server.port")
	if port != "" {
		panic(r.Run(":" + port))
	}
	panic(r.Run	())
}

func initConfig() {
	model.UserChannel = make(chan struct {}, 1000)
	model.GoodsChannel = make(chan struct {}, 1000)
	model.CartChannel = make(chan struct {}, 1000)

	workDir, _ := os.Getwd()
	viper.SetConfigName("application")
	viper.SetConfigType("yml")
	viper.AddConfigPath(workDir + "/config")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	common.InitFilter() //  初始化布隆过滤器用于过滤非法请求
}