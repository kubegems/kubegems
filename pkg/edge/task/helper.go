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

package task

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	edgev1beta1 "kubegems.io/kubegems/pkg/apis/edge/v1beta1"
)

func (r *Reconciler) UpdateEdgeTaskCondition(ctx context.Context, task *edgev1beta1.EdgeTask, condition edgev1beta1.EdgeTaskCondition) error {
	status := &task.Status
	index, oldcond := GetEdgeTaskCondition(status, condition.Type)
	now := metav1.Now()
	if oldcond == nil {
		condition.LastUpdateTime = now
		condition.LastTransitionTime = now
		status.Conditions = append(status.Conditions, condition)
	} else {
		if oldcond.Status != condition.Status {
			condition.LastTransitionTime = now
		} else {
			condition.LastTransitionTime = oldcond.LastTransitionTime
		}
		status.Conditions[index] = condition
	}
	if !reflect.DeepEqual(oldcond, condition) {
		if err := r.Client.Status().Update(ctx, task); err != nil {
			logr.FromContextOrDiscard(ctx).Error(err, "update edge task condition failed")
			return err
		}
	}
	return nil
}

func GetEdgeTaskCondition(status *edgev1beta1.EdgeTaskStatus, conditionType edgev1beta1.EdgeTaskConditionType) (int, *edgev1beta1.EdgeTaskCondition) {
	if status == nil {
		return -1, nil
	}
	for i, condition := range status.Conditions {
		if condition.Type == conditionType {
			return i, &condition
		}
	}
	return -1, nil
}

func RemoveEdgeTaskCondition(status *edgev1beta1.EdgeTaskStatus, conditionType edgev1beta1.EdgeTaskConditionType) {
	if status == nil {
		return
	}
	for i, condition := range status.Conditions {
		if condition.Type == conditionType {
			status.Conditions = append(status.Conditions[:i], status.Conditions[i+1:]...)
			return
		}
	}
}
