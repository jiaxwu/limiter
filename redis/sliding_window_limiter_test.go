package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

func TestNewSlidingWindowLimiter(t *testing.T) {
	type args struct {
		limit       int
		window      time.Duration
		smallWindow time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    *SlidingWindowLimiter
		wantErr bool
	}{
		{
			name: "60_5seconds",
			args: args{
				limit:       60,
				window:      time.Second * 5,
				smallWindow: time.Second,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{
				Addr: "127.0.0.1:6379",
			})
			l, err := NewSlidingWindowLimiter(client, tt.args.limit, tt.args.window, tt.args.smallWindow)
			if err != nil {
				t.Errorf("NewSlidingWindowLimiter() error = %v", err)
				return
			}
			successCount := 0
			for i := 0; i < tt.args.limit/2; i++ {
				if l.TryAcquire(context.Background(), "test") == nil {
					successCount++
				}
			}
			if successCount != tt.args.limit/2 {
				t.Errorf("NewSlidingWindowLimiter() got = %v, want %v", successCount, tt.args.limit/2)
				return
			}

			time.Sleep(time.Second * 2)
			successCount = 0
			for i := 0; i < tt.args.limit/3; i++ {
				if l.TryAcquire(context.Background(), "test") == nil {
					successCount++
				}
			}
			if successCount != tt.args.limit/3 {
				t.Errorf("NewSlidingWindowLimiter() got = %v, want %v", successCount, tt.args.limit/3)
			}

			time.Sleep(time.Second * 3)
			successCount = 0
			for i := 0; i < tt.args.limit; i++ {
				if l.TryAcquire(context.Background(), "test") == nil {
					successCount++
				}
			}
			if successCount != tt.args.limit/3*2 {
				t.Errorf("NewSlidingWindowLimiter() got = %v, want %v", successCount, tt.args.limit/3*2)
				return
			}
		})
	}
}
