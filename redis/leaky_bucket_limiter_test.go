package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

func TestNewLeakyBucketLimiter(t *testing.T) {
	type args struct {
		peakLevel       int
		currentVelocity int
	}
	tests := []struct {
		name    string
		args    args
		want    *LeakyBucketLimiter
		wantErr bool
	}{
		{
			name: "60",
			args: args{
				peakLevel:       60,
				currentVelocity: 10,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{
				Addr: "127.0.0.1:6379",
			})
			l := NewLeakyBucketLimiter(client, tt.args.peakLevel, tt.args.currentVelocity)
			successCount := 0
			for i := 0; i < tt.args.peakLevel*2; i++ {
				if l.TryAcquire(context.Background(), "test") == nil {
					successCount++
				}
			}
			if successCount != tt.args.peakLevel {
				t.Errorf("NewLeakyBucketLimiter() got = %v, want %v", successCount, tt.args.peakLevel)
				return
			}

			time.Sleep(time.Second * time.Duration(tt.args.peakLevel/tt.args.currentVelocity) / 2)
			successCount = 0
			for i := 0; i < tt.args.peakLevel; i++ {
				if l.TryAcquire(context.Background(), "test") == nil {
					successCount++
				}
			}
			if successCount != tt.args.peakLevel/2 {
				t.Errorf("NewLeakyBucketLimiter() got = %v, want %v", successCount, tt.args.peakLevel/2)
				return
			}
		})
	}
}
