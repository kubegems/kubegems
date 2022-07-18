// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
