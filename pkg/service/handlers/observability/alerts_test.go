package observability

import (
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"kubegems.io/kubegems/pkg/utils"
)

func Test_newDefaultSamplePair(t *testing.T) {
	type args struct {
		start time.Time
		end   time.Time
	}
	tests := []struct {
		name string
		args args
		want []model.SamplePair
	}{
		{
			name: "1",
			args: args{
				start: utils.DayStartTime(time.Now()),
				end:   utils.NextDayStartTime(time.Now()),
			},
			want: []model.SamplePair{
				{
					Timestamp: model.Time(utils.DayStartTime(time.Now()).Unix()),
					Value:     0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newDefaultSamplePair(tt.args.start, tt.args.end); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newDefaultSamplePair() = %v, want %v", got, tt.want)
			}
		})
	}
}
