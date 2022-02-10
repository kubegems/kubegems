package task

import (
	"fmt"
	"os"
	"time"

	"github.com/kubegems/gems/pkg/utils/workflow"
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
