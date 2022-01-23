package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

func TestNewTokenBucketLimiter(t *testing.T) {
	type args struct {
		capacity int
		rate     int
	}
	tests := []struct {
		name string
		args args
		want *TokenBucketLimiter
	}{
		{
			name: "60",
			args: args{
				capacity: 60,
				rate:     10,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{
				Addr: "127.0.0.1:6379",
			})
			l := NewTokenBucketLimiter(client, tt.args.capacity, tt.args.rate)
			successCount := 0
			for i := 0; i < tt.args.capacity; i++ {
				if l.TryAcquire(context.Background(), "test") == nil {
					successCount++
				}
			}
			if successCount != tt.args.capacity {
				t.Errorf("NewTokenBucketLimiter() got = %v, want %v", successCount, tt.args.capacity)
				return
			}

			time.Sleep(time.Second)
			successCount = 0
			for i := 0; i < tt.args.rate; i++ {
				if l.TryAcquire(context.Background(), "test") == nil {
					successCount++
				}
			}
			if successCount != tt.args.rate {
				t.Errorf("NewTokenBucketLimiter() got = %v, want %v", successCount, tt.args.rate)
				return
			}
		})
	}
}
