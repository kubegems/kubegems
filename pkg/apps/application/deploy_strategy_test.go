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

package application

import (
	"context"
	"reflect"
	"testing"

	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"k8s.io/utils/pointer"
)

func Test_completeCanarySteps(t *testing.T) {
	type args struct {
		in0    context.Context
		canary *rolloutsv1alpha1.CanaryStrategy
	}
	tests := []struct {
		name    string
		args    args
		want    *rolloutsv1alpha1.CanaryStrategy
		wantErr bool
	}{
		{
			name: "empty steps",
			args: args{canary: &rolloutsv1alpha1.CanaryStrategy{Steps: nil}},
			want: &rolloutsv1alpha1.CanaryStrategy{
				Steps: []rolloutsv1alpha1.CanaryStep{
					{SetWeight: pointer.Int32(defaultStepInitWeight)},
					{Pause: &rolloutsv1alpha1.RolloutPause{}},
				},
			},
		},
		{
			name: "init step weight",
			args: args{canary: &rolloutsv1alpha1.CanaryStrategy{Steps: []rolloutsv1alpha1.CanaryStep{
				{SetWeight: pointer.Int32(20)},
			}}},
			want: &rolloutsv1alpha1.CanaryStrategy{
				Steps: []rolloutsv1alpha1.CanaryStep{
					{SetWeight: pointer.Int32(20)},
					{Pause: &rolloutsv1alpha1.RolloutPause{}},
				},
			},
		},
		{
			name: "other steps will not change",
			args: args{canary: &rolloutsv1alpha1.CanaryStrategy{Steps: []rolloutsv1alpha1.CanaryStep{
				{SetWeight: pointer.Int32(20)},
				{Pause: &rolloutsv1alpha1.RolloutPause{}},
				{SetWeight: pointer.Int32(80)},
			}}},
			want: &rolloutsv1alpha1.CanaryStrategy{
				Steps: []rolloutsv1alpha1.CanaryStep{
					{SetWeight: pointer.Int32(20)},
					{Pause: &rolloutsv1alpha1.RolloutPause{}},
					{SetWeight: pointer.Int32(80)},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := completeCanarySteps(tt.args.in0, tt.args.canary); (err != nil) != tt.wantErr {
				t.Errorf("completeCanarySteps() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.canary, tt.want) {
				t.Errorf("completeCanarySteps() changed = %v, want %v", tt.args.canary, tt.want)
			}
		})
	}
}
