

-- 删除商品信息的缓存
-- KEYS[1] = ID KEYS[2] = email KEYS[3] = type
redis.call('LREM', "GoodsIDList"..KEYS[2], "0", KEYS[1]) -- 删掉用户商品ID队列中的ID
redis.call('DEL', "GoodsID"..KEYS[1]) -- 删掉商品详细信息
local zsetKey = ""
if KEYS[3] == "1" then
    zsetKey = "lostZset"
elseif KEYS[3] == "2" then
    zsetKey = "electronicZset"
elseif KEYS[3] == "3" then
    zsetKey = "liveZset"
elseif KEYS[3] == "4" then
    zsetKey = "foodZset"
end
redis.call('ZREM', zsetKey, KEY[1]) -- 删除zset中的ID
--[[

1. 用户信息 数据结构是Hash KEY = 「1522972330@qq.com」 用户邮箱
FIELD ： name telephone sid password class signature

2. 用户信息修改的分布式锁 数据结构式是String KEY = 「lock1522972330@qq.com」 lock+UserEmail
Note：进行用户信息修改前先通过lua脚本原子性的将该值设置为1，并且设置过期时间200ms，本次修改完之后会主动尝试删除该值

3. 为了实现幂等性,对于同一IP限制访问频率 数据结构是String KEY = 「IP127.0.0.1」 IP+UserIP VALUE = 最后一次访问成功的请求Unix毫秒时间
Note: 在中间件OnceRequest.go中使用，每次访问请求先获取该IP上次访问的时间,最快两次访问频率需要大于300ms

4. 用户所发布的商品 数据结构是 LIST  KEY = 「GoodsIDList1522972330@qq.com」 GoodsIDList+UserEmail
Note： Value是所发布商品的在MYSQL存储中的ID

5. 商品的所有信息 数据结构是 Hash KEY = 「GoodsID123」 GoodsID+GoodsID
FIELD： name price wordDetail mainPicture morePicture type ownerEmail

6. 为了实现分页取数据 按照商品分类存入Zset按照ID排序 KEY = 「lostZset」 lostZset
member = GoodsID
score = GoodsID

7. 商品信息修改的分布式锁 数据结构是String KEY = 「goodsLock123」 goodsLock+GoodsID
Note: 细分到某一件商品上锁,可以提高并发

8. 用户购物车内容 数据结构是 LIST KEY = 「Cart:1522972330@qq.com」 Cart:+OwnerEmail
Note：只记录GoodsID，再到GoodsID查询商品所有信息即可

9. 购物车信息修改的分布式锁 String; KEY = 「cartLock1522972330@qq.com」 cartLock+UserEmail

--]]

--[[
-- 写入个人信息到redis KEY = email, 并且释放掉锁
redis.call('HSET', KEYS[1], "name", ARGV[1]);
redis.call('HSET', KEYS[1], "telephone", ARGV[2]);
redis.call('HSET', KEYS[1], "sid", ARGV[3]);
redis.call('HSET', KEYS[1], "password", ARGV[4]);
redis.call('HSET', KEYS[1], "class", ARGV[5]);
redis.call('HSET', KEYS[1], 'signature', ARGV[6])
if redis.call('EXISTS', KEYS[2]) == 1 then
    redis.call('DEL', KEYS[2])
end

-- 用户注册检测是否有锁 Goods可复用
local t = redis.call('EXISTS', KEYS[1])
if t == 0 then
    redis.call('SET', KEYS[1], 1)
    redis.call('PEXPIRE', KEYS[1], 100) -- 过期时间的设定需要考虑实际业务速度
    return 1
end
return 0

-- 幂等性
if redis.call('EXISTS', KEYS[1]) == 0 then -- 不存在此IP的访问记录则设置上去
    redis.call('SET', KEYS[1], ARGV[1])
    return 1
end
local lasttime = redis.call('GET', KEYS[1])
if ARGV[1] - lasttime > 30000 then             -- 超过300ms
    redis.call('SET', KEYS[1], ARGV[1])
    return 1
end
return 0
--]]

--[[
---- 从redis获取用户信息
local ans = {}
ans[#ans+1] = redis.call('HGET', KEYS[1], "name")
ans[#ans+1] = redis.call('HGET', KEYS[1], 'telephone')
ans[#ans+1] = redis.call('HGET', KEYS[1], 'sid')
ans[#ans+1] = redis.call('HGET', KEYS[1], 'password')
ans[#ans+1] = redis.call('HGET', KEYS[1], 'class')
ans[#ans+1] = redis.call('HGET', KEYS[1], 'signature')
return ans
--]]

--[[
-- 更新数据到redis 需要判断空字符串
if ARGV[1] ~= "" then
    redis.call('HSET', KEYS[1], "name", ARGV[1])
end
if ARGV[2] ~= "" then
    redis.call('HSET', KEYS[1], "telephone", ARGV[2])
end
if ARGV[3] ~= "" then
    redis.call('HSET', KEYS[1], "sid", ARGV[3])
end
if ARGV[4] ~= "" then
    redis.call('HSET', KEYS[1], "class", ARGV[4])
end
if ARGV[5] ~= "" then
    redis.call('HSET', KEYS[1], "signature", ARGV[5])
end
--]]



--[[
-- 存储商品信息 key = "GoodsIDList" + userEmail ARGV[1] = Goods.ID
local key1 = "GoodsIDList" .. KEYS[1]
redis.call('LPUSH', key1, ARGV[1])
local key = "GoodsID" .. ARGV[1]
redis.call('HSET', key, 'name', ARGV[2])
redis.call('HSET', key, 'price', ARGV[3])
redis.call('HSET', key, 'wordDetail', ARGV[4])
redis.call('HSET', key, 'mainPicture', ARGV[5])
redis.call('HSET', key, 'morePicture', ARGV[6])
redis.call('HSET', key, 'type', ARGV[7])
redis.call('HSET', key, 'email', KEYS[1])
-- ZADD key Goods.ID Goods.Id
-- 通过zset实现分页获取商品数据
if ARGV[7] == 1 then -- 丢失物品
    redis.call('ZADD', "lostZset", ARGV[1] * -1, "GoodsID")
elseif ARGV[7] == 2 then -- 电子商品
    redis.call('ZADD', "electronicZset", ARGV[1] * -1, "GoodsID")
elseif ARGV[7] == 3 then -- 生活用品
    redis.call('ZADD', "liveZset", ARGV[1] * -1, "GoodsID")
elseif ARGV[7] == 4 then -- 食品
    redis.call('ZADD', "foodZset", ARGV[1] * -1, "GoodsID")
end
--]]


---- 分页获取商品信息 先从zset中拿到Id，再找到所有数据 只需要取出 名称 + 价格
---- KEYS[1] = type
local type = KEYS[1]
local start = KEYS[2]
local stop = KEYS[3]
local IDs = {}
if type == "1" then
    IDs = redis.call('ZREVRANGE', 'lostZset', start, stop)
elseif type == "2" then
    IDs = redis.call('ZREVRANGE', 'electronicZset', start, stop)
elseif type == "3" then
    IDs = redis.call('ZREVRANGE', 'liveZset', start, stop)
elseif type == "4" then
    IDs = redis.call('ZREVRANGE', 'foodZset', start, stop)
end
local res = {}
for i = 1, #IDs do
    local ID = IDs[i]
    local name = redis.call('HGET', "GoodsID" .. ID, "name")
    local price = redis.call('HGET', "GoodsID" .. ID, "price")
    local mainPicture = redis.call('HGET', "GoodsID" .. ID, "mainPicture")
    res[#res+1] = name .. "," .. price .. "," .. mainPicture
end
return res


--[[
 通过Email遍历队列获取该用户所有商品ID，然后找到每个商品的所有信息，通过逗号分割，go再通过分割字符串处理
local ids = redis.call('LRANGE', "GoodsIDList" .. KEYS[1], 0, -1)
local res = {}
for i = 1, #ids do
    local idKey = "GoodsID" .. ids[i]
    local name = redis.call('HGET', idKey, 'name')
    local price = redis.call('HGET', idKey, 'price')
    local wordDetail = redis.call('HGET', idKey, 'wordDetail')
    local mainPicture = redis.call('HGET', idKey, 'mainPicture')
    local morePicture = redis.call('HGET', idKey, 'morePicture')
    local type = redis.call('HGET', idKey, 'type')
    res[#res+1] = name..","..ids[i]..","..price..","..wordDetail..","..mainPicture..","..morePicture..","..type
end
return res
--]]

--[[
-- 更新goods信息到redis
if ARGV[1] ~= "" then
    redis.call('HSET', KEYS[1], "name", ARGV[1])
end
if ARGV[2] ~= "" then
    redis.call('HSET', KEYS[1], "price", ARGV[2])
end
if ARGV[3] ~= "" then
    redis.call('HSET', KEYS[1], "type", ARGV[3])
end
if ARGV[4] ~= "" then
    redis.call('HSET', KEYS[1], "mainPicture", ARGV[4])
end
if ARGV[5] ~= "" then
    redis.call('HSET', KEYS[1], "morePicture", ARGV[5])
end
if ARGV[6] ~= "" then
    redis.call('HSET', KEYS[1], "wordDetail", ARGV[6])
end
--]]

--[[
-- 删除商品信息的缓存
-- KEYS[1] = ID KEYS[2] = email KEYS[3] = type
redis.call('LREM', "GoodsIDList"..KEYS[2], "0", KEYS[1]) -- 删掉用户商品ID队列中的ID
redis.call('DEL', "GoodsID"..KEYS[1]) -- 删掉商品详细信息
local zsetKey = ""
if KEYS[3] == '1' then
    zsetKey = "lostZset"
elseif KEYS[3] == "2" then
    zsetKey = "electronicZset"
elseif KEYS[3] == "3" then
    zsetKey = "liveZset"
elseif KEYS[3] == "4" then
    zsetKey = "foodZset"
end
redis.call('ZREM', zsetKey, KEY[1]) -- 删除zset中的ID
--]]


--[[
-- 显示商品详情 KEYS[1] = goodsID
local name = redis.call('HGET', 'GoodsID' .. KEYS[1], "name")
local price = redis.call('HGET', 'GoodsID' .. KEYS[1], "price")
local type = redis.call('HGET', 'GoodsID' .. KEYS[1], "type")
local mainPicture = redis.call('HGET', 'GoodsID' .. KEYS[1], "mainPicture")
local morePicture = redis.call('HGET', 'GoodsID' .. KEYS[1], "morePicture")
local wordDetail = redis.call('HGET', 'GoodsID' .. KEYS[1], "wordDetail")
local email = redis.call('HGET', 'GoodsID' .. KEYS[1], "email")
return name .. "," .. price .. "," .. type .. "," .. mainPicture .. "," .. morePicture .. "," .. wordDetail .. "," .. email

 插入购物车信息 type = queue KEYS[1] = email     key = Cart:EMAIL
 LPUSH key value
--]]

--[[
-- 获取购物车信息 从队列中拿到goodsID再拿数据 KEYS[1] = ownerEmail
local key = "Cart:" .. KEYS[1]
local ids = redis.call('LRANGE', key, 0, -1)
local ret = {}
for i = 1, #ids do
    local GoodsKey = "GOodsID" .. ids[i]
    local name = redis.call('HGET', GoodsKey, 'name')
    local price = redis.call('HGET', GoodsKey, 'price')
    local mainPicture = redis.call('HGET', GoodsKey, 'mainPicture')
    ret[#ret+1] = ids[i]..","..name..","..price..","..mainPicture
end
return ret
--]]

if KEYS[1] == 1 or 1 then
    return
end

local t = redis.call('EXISTS', KEYS[1])
    if t == 0 then
        redis.call('SET', KEYS[1], 1)
        redis.call('PEXPIRE', KEYS[1], 100) -- 过期时间的设定需要考虑实际业务速度
    return 1
end
return 0