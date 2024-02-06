// Copyright 2023 The kubegems.io Authors
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

package auth

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
)

const DefaultAuditLogCacheSize = 256

func NewCachedAuditSink(ctx context.Context, sink AuditSink, maxCacheSize int) AuditSink {
	if maxCacheSize <= 0 {
		maxCacheSize = DefaultAuditLogCacheSize
	}
	logger := logr.FromContextOrDiscard(ctx).WithName("cached-audit-sink")
	cachesink := &CachedAuditSink{
		sink:   sink,
		cache:  make(chan *AuditLog, maxCacheSize),
		logger: logger,
	}
	go func() {
		for {
			select {
			case auditlog := <-cachesink.cache:
				if err := sink.Save(auditlog); err != nil {
					logger.Error(err, "save audit log")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return cachesink
}

type CachedAuditSink struct {
	sink   AuditSink
	logger logr.Logger
	cache  chan *AuditLog
}

func (c *CachedAuditSink) Save(log *AuditLog) error {
	select {
	case c.cache <- log:
	default:
		c.logger.Error(fmt.Errorf("cache is full"), "save audit log")
		return fmt.Errorf("cache is full")
	}
	return nil
}
