create table userInfo (
    ID INT PRIMARY KEY AUTO_INCREMENT comment '用户主键',
    userName VARCHAR(100) NOT NULL comment '用户姓名',
    userEmail VARCHAR(100) NOT NULL UNIQUE comment '用户邮箱',
    userTelephone VARCHAR(15) NOT NULL UNIQUE comment '用户手机号',
    userSid VARCHAR(15) NOT NULL UNIQUE comment '用户学号，也用来当作账号',
    userPassword VARCHAR(15) NOT NULL comment '用户密码',
    userSignature VARCHAR(100) default '该用户没有留下任何签名' comment '用户个性签名',
    userClass VARCHAR(50) default '未知班级' comment '用户班级',
    deleted INT default 0 comment '删除标记',
    CreatedAt TIME comment '',
    UpdatedAt Time,
    DeletedAt Time
) comment '用户信息';


create table Cart (
    ID INT PRIMARY KEY AUTO_INCREMENT comment '购物车主键',
    userEmail VARCHAR(15) NOT NULL UNIQUE comment '用户邮箱',
    userId INT NOT NULL UNIQUE comment 'userInfo主键id',
    lostId INT NOT NULL UNIQUE comment '丢失商品的主键id',
    goodsId INT NOT NULL UNIQUE comment '二手商品的主键id',
    deleted INT default 0 comment '删除标记',

    CreatedAt TIME comment '创建时间',
    UpdatedAt Time comment '修改时间',
    DeletedAt Time comment '删除时间'
) comment '购物车';

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

create table lostGoods (
    ID INT PRIMARY KEY AUTO_INCREMENT comment '丢失物品的主键',
    lostName VARCHAR(300) NOT NULL default '未命名' comment '丢失商品',
    lostSid VARCHAR(15) default '' comment '丢失校园卡学号',
    lostDetail VARCHAR(1000) default '' comment '丢失物品的详细信息',
    deleted INT default 0 comment '删除标记',
    CreatedAt TIME comment '创建时间',
    UpdatedAt Time comment '修改时间',
    DeletedAt Time comment '删除时间'
) comment '丢失物品专区';