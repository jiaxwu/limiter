package main

import (
	"fmt"
	"github.com/jiaxwu/limiter"
	"net/http"
	"time"
)

func main() {
	l, _ := limiter.NewSlidingLogLimiter(time.Second,
		limiter.NewSlidingLogLimiterStrategy(10, time.Second*30),
		limiter.NewSlidingLogLimiterStrategy(15, time.Minute))
	count := 0
	http.HandleFunc("/_sliding_log_limiter_test", func(w http.ResponseWriter, r *http.Request) {
		err := l.TryAcquire()
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
