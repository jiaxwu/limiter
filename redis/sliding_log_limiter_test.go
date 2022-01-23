package redis

import (
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

func TestNewSlidingLogLimiter(t *testing.T) {
	type args struct {
		smallWindow time.Duration
		strategies  []*SlidingLogLimiterStrategy
	}
	tests := []struct {
		name    string
		args    args
		want    *SlidingLogLimiter
		wantErr bool
	}{
		{
			name: "60_5seconds",
			args: args{
				smallWindow: time.Second,
				strategies: []*SlidingLogLimiterStrategy{
					NewSlidingLogLimiterStrategy(10, time.Minute),
					NewSlidingLogLimiterStrategy(100, time.Hour),
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{
				Addr: "127.0.0.1:6379",
			})
			NewSlidingLogLimiter(client, tt.args.smallWindow, tt.args.strategies...)
		})
	}
}
