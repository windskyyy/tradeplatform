package model

import "github.com/jinzhu/gorm"

var GoodsChannel chan struct {}

type Goods struct {
	gorm.Model
	GoodsID int `form:"GoodsID"`
	Name string `gorm:"type:VARCHAR(100);not null"`
	Price float64 `gorm:"type:float;not null"`
	WordDetail string `gorm:"type:VARCHAR(500);not null"`
	MainPicture string `gorm:"type:VARCHAR(500); not null"`
	MorePictures string `gorm:"type:TEXT;"`
	Type int `gorm:"type:INT"`
	OwnerEmail string `gorm:"type:VARCHAR(50); not null"`
	Deleted string `gorm:"type:VARCHAR(10)"`
}

/*
create table Goods (
    ID INT PRIMARY KEY AUTO_INCREMENT comment '二手商品的主键',
    Name VARCHAR(100) NOT NULL comment '商品名称',
    Price FLOAT NOT NULL comment '商品价格',
    WorkDetail VARCHAR(500) comment '商品文字详情',
    MainPicture VARCHAR(500) comment '主图片',
    MorePictures VARCHAR(500) comment '更多图片展示',
    Type INT comment '商品分类，1->丢失物品，2->电子商品，3->生活用品，4->食品',
    deleted INT default 0 comment '删除标记',

    CreatedAt TIME comment '创建时间',
    UpdatedAt Time comment '修改时间',
    DeletedAt Time comment '删除时间'
) comment '二手商品';
 */