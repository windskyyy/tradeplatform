package model

import "github.com/jinzhu/gorm"

var CartChannel chan struct {}

type Cart struct {
	gorm.Model
	GoodsID uint `gorm:"type:int unsigned"`
	Name string `gorm:"type:VARCHAR(100); not null"`
	Price float64 `gorm:"type:float; not null"`
	MainPicture string `gorm:"type:VARCHAR(100); not null"`
	OwnerEmail string `gorm:"type:VARCHAR(20); not null"`
	deleted int `gorm:"type:int"`
}
