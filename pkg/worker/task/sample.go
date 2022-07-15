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

package task

import (
	"fmt"
	"os"
	"time"

	"kubegems.io/kubegems/pkg/utils/workflow"
)

type SampleTasker struct{}

func (s *SampleTasker) ProvideFuntions() map[string]interface{} {
	return map[string]interface{}{
		"now": func() string {
			return time.Now().Format(time.RFC3339)
		},
		"hostname": func() string {
			hostname, _ := os.Hostname()
			return hostname
		},
		"greet": func(name string) string {
			return fmt.Sprintf("hello %s,nice to see you", name)
		},
	}
}

func (s *SampleTasker) Crontasks() map[string]Task {
	return map[string]Task{
		"@daily": {
			Name:  "daily-greet",
			Group: "sample",
			Steps: []workflow.Step{
				{
					Name:     "see-now",
					Function: "now",
				},
				{
					Name:     "say-hi",
					Function: "greet",
					Args:     workflow.ArgsOf("jackson"),
				},
			},
		},
	}
}
