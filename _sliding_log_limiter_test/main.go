package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	limiter "github.com/jiaxwu/limiter/redis"
	"net/http"
	"time"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	l, _ := limiter.NewSlidingLogLimiter(client, time.Second,
		limiter.NewSlidingLogLimiterStrategy(10, time.Second*30),
		limiter.NewSlidingLogLimiterStrategy(15, time.Minute))
	count := 0
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		err := l.TryAcquire(context.Background(), "test")
		if err == nil {
			w.Write([]byte("请求成功" + time.Now().String()))
			count++
			fmt.Println(count)
		} else {
			w.Write([]byte(err.Error() + time.Now().String()))
		}
	})
	http.ListenAndServe("127.0.0.1:8080", nil)
}
