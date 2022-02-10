package retry

import (
	"context"
	"errors"
	"math"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

var DefaultBackoff = wait.Backoff{
	Steps:    math.MaxInt32,   // 最大重试次数
	Duration: 5 * time.Second, // 每次重试的基准间隔时间
	Factor:   1.1,             // 每次重试的倍率，在前一次的时间上 * factor 为新一次的重试等待时间
	Jitter:   0.1,             // 抖动
}

func AlwaysError(err error) bool { return true }

func Always(fn func() error) error {
	return retry.OnError(DefaultBackoff, AlwaysError, fn)
}

func OnError(isRetry func(error) bool, fn func() error) error {
	return retry.OnError(DefaultBackoff, isRetry, fn)
}

func NotContextCancelError(err error) bool {
	return !errors.Is(err, context.Canceled)
}
