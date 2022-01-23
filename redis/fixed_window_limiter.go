package redis

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

const fixedWindowLimiterTryAcquireRedisScript = `
-- ARGV[1]: 窗口时间大小
-- ARGV[2]: 窗口请求上限

local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])

-- 获取原始值
local counter = tonumber(redis.call("get", KEYS[1]))
if counter == nil then 
	counter = 0
end
-- 若到达窗口请求上限，请求失败
if counter >= limit then
	return 0
end
-- 窗口值+1
redis.call("incr", KEYS[1])
if counter == 0 then
    redis.call("pexpire", KEYS[1], window)
end
return 1
`

// FixedWindowLimiter 固定窗口限流器
type FixedWindowLimiter struct {
	limit  int           // 窗口请求上限
	window int           // 窗口时间大小
	client *redis.Client // Redis客户端
	script *redis.Script // TryAcquire脚本
}

func NewFixedWindowLimiter(client *redis.Client, limit int, window time.Duration) (*FixedWindowLimiter, error) {
	// redis过期时间精度最大到毫秒，因此窗口必须能被毫秒整除
	if window%time.Millisecond != 0 {
		return nil, errors.New("the window uint must not be less than millisecond")
	}

	return &FixedWindowLimiter{
		limit:  limit,
		window: int(window / time.Millisecond),
		client: client,
		script: redis.NewScript(fixedWindowLimiterTryAcquireRedisScript),
	}, nil
}

func (l *FixedWindowLimiter) TryAcquire(ctx context.Context, resource string) error {
	success, err := l.script.Run(ctx, l.client, []string{resource}, l.window, l.limit).Bool()
	if err != nil {
		return err
	}
	// 若到达窗口请求上限，请求失败
	if !success {
		return ErrAcquireFailed
	}
	return nil
}
