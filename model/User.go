package model

import "github.com/jinzhu/gorm"

var UserChannel chan struct {}

type User struct {
	gorm.Model
	UserName string `gorm:"type:varchar(100);not null"`
	UserEmail string `gorm:"type:varchar(50);not null;unique"`
	UserTelephone string `gorm:"varchar(15);not null;unique"`
	UserSid string `gorm:"type:varchar(15);not null; unique"`
	UserPassword string `gorm:"type:varchar(30);not null"`
	UserClass string `gorm:"type:varchar(50);default:'未知班级'"`
	UserSignature string `gorm:"type:varchar(200);default:'该用户没留下任何签名'"`
	UserNewPassword string `gorm:"type:varchar(30);not null"`
	Deleted int `gorm:"type:int"`
}

/*
	ID INT PRIMARY KEY AUTO_INCREMENT comment '用户主键',
    userName VARCHAR(100) NOT NULL comment '用户姓名',
    userEmail VARCHAR(100) NOT NULL UNIQUE comment '用户邮箱',
    userTelephone VARCHAR(15) NOT NULL UNIQUE comment '用户手机号',
    userSid VARCHAR(15) NOT NULL UNIQUE comment '用户学号，也用来当作账号',
    userPassword VARCHAR(15) NOT NULL comment '用户密码',
    userSignature VARCHAR(100) default '该用户没有留下任何签名' comment '用户个性签名',
    userClass VARCHAR(50) default '未知班级' comment '用户班级'
 */