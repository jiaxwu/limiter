package redis

import "errors"

// ErrAcquireFailed 获取失败
var ErrAcquireFailed = errors.New("acquire failed")
