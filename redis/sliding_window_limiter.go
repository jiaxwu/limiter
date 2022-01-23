package redis

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

const slidingWindowLimiterTryAcquireRedisScriptHashImpl = `
-- ARGV[1]: 窗口时间大小
-- ARGV[2]: 窗口请求上限
-- ARGV[3]: 当前小窗口值
-- ARGV[4]: 起始小窗口值

local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local currentSmallWindow = tonumber(ARGV[3])
local startSmallWindow = tonumber(ARGV[4])

-- 计算当前窗口的请求总数
local counters = redis.call("hgetall", KEYS[1])
local count = 0
for i = 1, #(counters) / 2 do 
	local smallWindow = tonumber(counters[i * 2 - 1])
	local counter = tonumber(counters[i * 2])
	if smallWindow < startSmallWindow then
		redis.call("hdel", KEYS[1], smallWindow)
	else 
		count = count + counter
	end
end

-- 若到达窗口请求上限，请求失败
if count >= limit then
	return 0
end

-- 若没到窗口请求上限，当前小窗口计数器+1，请求成功
redis.call("hincrby", KEYS[1], currentSmallWindow, 1)
redis.call("pexpire", KEYS[1], window)
return 1
`

const slidingWindowLimiterTryAcquireRedisScriptListImpl = `
-- ARGV[1]: 窗口时间大小
-- ARGV[2]: 窗口请求上限
-- ARGV[3]: 当前小窗口值
-- ARGV[4]: 起始小窗口值

local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local currentSmallWindow = tonumber(ARGV[3])
local startSmallWindow = tonumber(ARGV[4])

-- 获取list长度
local len = redis.call("llen", KEYS[1])
-- 如果长度是0，设置counter，长度+1
local counter = 0
if len == 0 then 
	redis.call("rpush", KEYS[1], 0)
	redis.call("pexpire", KEYS[1], window)
	len = len + 1
else
	-- 如果长度大于1，获取第二第个元素
	local smallWindow1 = tonumber(redis.call("lindex", KEYS[1], 1))
	counter = tonumber(redis.call("lindex", KEYS[1], 0))
	-- 如果该值小于起始小窗口值
	if smallWindow1 < startSmallWindow then 
		local count1 = redis.call("lindex", KEYS[1], 2)
		-- counter-第三个元素的值
		counter = counter - count1
		-- 长度-2
		len = len - 2
		-- 删除第二第三个元素
		redis.call("lrem", KEYS[1], 1, smallWindow1)
		redis.call("lrem", KEYS[1], 1, count1)
	end
end

-- 若到达窗口请求上限，请求失败
if counter >= limit then 
	return 0
end 

-- 如果长度大于1，获取倒数第二第一个元素
if len > 1 then
	local smallWindown = tonumber(redis.call("lindex", KEYS[1], -2))
	-- 如果倒数第二个元素小窗口值大于等于当前小窗口值
	if smallWindown >= currentSmallWindow then
		-- 把倒数第二个元素当成当前小窗口（因为它更新），倒数第一个元素值+1
		local countn = redis.call("lindex", KEYS[1], -1)
		redis.call("lset", KEYS[1], -1, countn + 1)
	else 
		-- 否则，添加新的窗口值，添加新的计数（1），更新过期时间
		redis.call("rpush", KEYS[1], currentSmallWindow, 1)
		redis.call("pexpire", KEYS[1], window)
	end
else 
	-- 否则，添加新的窗口值，添加新的计数（1），更新过期时间
	redis.call("rpush", KEYS[1], currentSmallWindow, 1)
	redis.call("pexpire", KEYS[1], window)
end 

-- counter + 1并更新
redis.call("lset", KEYS[1], 0, counter + 1)
return 1
`

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	limit        int           // 窗口请求上限
	window       int64         // 窗口时间大小
	smallWindow  int64         // 小窗口时间大小
	smallWindows int64         // 小窗口数量
	client       *redis.Client // Redis客户端
	script       *redis.Script // TryAcquire脚本
}

func NewSlidingWindowLimiter(client *redis.Client, limit int, window, smallWindow time.Duration) (
	*SlidingWindowLimiter, error) {
	// redis过期时间精度最大到毫秒，因此窗口必须能被毫秒整除
	if window%time.Millisecond != 0 || smallWindow%time.Millisecond != 0 {
		return nil, errors.New("the window uint must not be less than millisecond")
	}

	// 窗口时间必须能够被小窗口时间整除
	if window%smallWindow != 0 {
		return nil, errors.New("window cannot be split by integers")
	}

	return &SlidingWindowLimiter{
		limit:        limit,
		window:       int64(window / time.Millisecond),
		smallWindow:  int64(smallWindow / time.Millisecond),
		smallWindows: int64(window / smallWindow),
		client:       client,
		script:       redis.NewScript(slidingWindowLimiterTryAcquireRedisScriptListImpl),
	}, nil
}

func (l *SlidingWindowLimiter) TryAcquire(ctx context.Context, resource string) error {
	// 获取当前小窗口值
	currentSmallWindow := time.Now().UnixMilli() / l.smallWindow * l.smallWindow
	// 获取起始小窗口值
	startSmallWindow := currentSmallWindow - l.smallWindow*(l.smallWindows-1)

	success, err := l.script.Run(
		ctx, l.client, []string{resource}, l.window, l.limit, currentSmallWindow, startSmallWindow).Bool()
	if err != nil {
		return err
	}
	// 若到达窗口请求上限，请求失败
	if !success {
		return ErrAcquireFailed
	}
	return nil
}
