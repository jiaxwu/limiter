package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

const leakyBucketLimiterTryAcquireRedisScript = `
-- ARGV[1]: 最高水位
-- ARGV[2]: 水流速度/秒
-- ARGV[3]: 当前时间（秒）

local peakLevel = tonumber(ARGV[1])
local currentVelocity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local lastTime = tonumber(redis.call("hget", KEYS[1], "lastTime"))
local currentLevel = tonumber(redis.call("hget", KEYS[1], "currentLevel"))
-- 初始化
if lastTime == nil then 
	lastTime = now
	currentLevel = 0
	redis.call("hmset", KEYS[1], "currentLevel", currentLevel, "lastTime", lastTime)
end 

-- 尝试放水
-- 距离上次放水的时间
local interval = now - lastTime
if interval > 0 then
	-- 当前水位-距离上次放水的时间(秒)*水流速度
	local newLevel = currentLevel - interval * currentVelocity
	if newLevel < 0 then 
		newLevel = 0
	end 
	currentLevel = newLevel
	redis.call("hmset", KEYS[1], "currentLevel", newLevel, "lastTime", now)
end

-- 若到达最高水位，请求失败
if currentLevel >= peakLevel then
	return 0
end
-- 若没有到达最高水位，当前水位+1，请求成功
redis.call("hincrby", KEYS[1], "currentLevel", 1)
redis.call("expire", KEYS[1], peakLevel / currentVelocity)
return 1
`

// LeakyBucketLimiter 漏桶限流器
type LeakyBucketLimiter struct {
	peakLevel       int           // 最高水位
	currentVelocity int           // 水流速度/秒
	client          *redis.Client // Redis客户端
	script          *redis.Script // TryAcquire脚本
}

func NewLeakyBucketLimiter(client *redis.Client, peakLevel, currentVelocity int) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		peakLevel:       peakLevel,
		currentVelocity: currentVelocity,
		client:          client,
		script:          redis.NewScript(leakyBucketLimiterTryAcquireRedisScript),
	}
}

func (l *LeakyBucketLimiter) TryAcquire(ctx context.Context, resource string) error {
	// 当前时间
	now := time.Now().Unix()
	success, err := l.script.Run(ctx, l.client, []string{resource}, l.peakLevel, l.currentVelocity, now).Bool()
	if err != nil {
		return err
	}
	// 若到达窗口请求上限，请求失败
	if !success {
		return ErrAcquireFailed
	}
	return nil
}
