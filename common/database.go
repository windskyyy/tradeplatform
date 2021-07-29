package common

import (
	"TradePlatform/model"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"log"
	"net/url"
)

var db *gorm.DB
var pool *redis.Pool

func InitDB() {
	// mysql
	driverName := viper.GetString("datasource.driverName")
	host := viper.GetString("datasource.host")
	port := viper.GetString("datasource.port")
	database := viper.GetString("datasource.database")
	username := viper.GetString("datasource.username")
	password := viper.GetString("datasource.password")
	charset := viper.GetString("datasource.charset")
	loc := viper.GetString("datasource.loc")
	args := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=true&loc=%s",
		username,
		password,
		host,
		port,
		database,
		charset,
		url.QueryEscape(loc))

	var err error
	db, err = gorm.Open(driverName, args)
	if err != nil {
		panic("failed to connect database, err: " + err.Error())
	}
	db.DB().SetMaxIdleConns(100)
	db.DB().SetMaxOpenConns(1000)
	// 初始化表
	db.AutoMigrate(&model.User{})
	db.AutoMigrate(&model.Goods{})
	db.AutoMigrate(&model.Cart{})
	//db.DB().SetMaxIdleConns(0)

	// redis连接池
	pool = &redis.Pool{     //实例化一个连接池
		MaxIdle: 512,    //最初的连接数量
		MaxActive: 0,    //连接池最大连接数量,不确定可以用0（0表示自动定义），按需分配
		IdleTimeout: 30,    //连接关闭时间
		Dial: func() (redis.Conn ,error){     //要连接的redis数据库
			return redis.Dial("tcp","127.0.0.1:6379")
		},
	}
}

func GetRedis() redis.Conn {
	return pool.Get()
}

func GetDB() *gorm.DB {
	return db
}

func CloseDB() {
	err := db.Close()
	if err != nil {
		log.Println("DB err = ", err)
	}
}
