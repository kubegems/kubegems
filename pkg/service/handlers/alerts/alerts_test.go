package alerthandler

import (
	"reflect"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/common/model"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/prometheus"
)

func TestAlertRule_checkAndModify(t *testing.T) {
	type fields struct {
		Name           string
		Namespace      string
		Rules          []monitoringv1.Rule
		Receivers      []Receiver
		IsOpen         *bool
		PromeAlertRule *prometheus.RealTimeAlertRule
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "null receiver",
			fields: fields{
				Receivers: []Receiver{},
			},
			wantErr: true,
		},
		{
			name: "repeat receiver",
			fields: fields{
				Receivers: []Receiver{
					{
						Name:     "rec-1",
						Interval: "10s",
					},
					{
						Name:     "rec-1",
						Interval: "20s",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no repeat receiver",
			fields: fields{
				Receivers: []Receiver{
					{
						Name:     "rec-1",
						Interval: "10s",
					},
					{
						Name:     "rec-2",
						Interval: "20s",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// r := &AlertRule{
			// 	Name:           tt.fields.Name,
			// 	Namespace:      tt.fields.Namespace,
			// 	Rules:          tt.fields.Rules,
			// 	Receivers:      tt.fields.Receivers,
			// 	IsOpen:         tt.fields.IsOpen,
			// 	PromeAlertRule: tt.fields.PromeAlertRule,
			// }
			// if err := r.; (err != nil) != tt.wantErr {
			// 	t.Errorf("AlertRule.checkAndModify() error = %v, wantErr %v", err, tt.wantErr)
			// }
		})
	}
}

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
