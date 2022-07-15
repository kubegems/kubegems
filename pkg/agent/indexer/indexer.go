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

package indexer

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// 自定义的indexer
func CustomIndexPods(c cache.Cache) error {
	//  1. 根据状态(kubectl看到的)过滤
	if err := c.IndexField(context.TODO(), &v1.Pod{}, "phase", func(iobj client.Object) []string {
		obj := iobj.(*v1.Pod)
		return []string{podStatus(obj)}
	}); err != nil {
		return err
	}

	// 2. 根据节点过滤
	if err := c.IndexField(context.TODO(), &v1.Pod{}, "nodename", func(iobj client.Object) []string {
		obj := iobj.(*v1.Pod)
		return []string{obj.Spec.NodeName}
	}); err != nil {
		return err
	}

	return nil
}

func podStatus(po *v1.Pod) string {
	/*
		NOTICE: 这儿修改一定要和前端保持一致的逻辑
			根据pod生命周期，pod的生命周期分为 Pending, Running, Succeeded, Failed, Unknow 五个大状态
			容器又分为三种大状态 Waiting, Running, Terminated
			以上，容器真实状态判断函数如下
	*/
	if po.GetDeletionTimestamp() != nil {
		return "Terminating"
	}

	if len(po.Status.ContainerStatuses) == 0 {
		if len(po.Status.Reason) > 0 {
			return po.Status.Reason
		} else {
			return string(po.Status.Phase)
		}
	}
	st := "Running"
	for _, co := range po.Status.ContainerStatuses {
		if co.State.Waiting != nil {
			st = co.State.Waiting.Reason
		} else if co.State.Terminated != nil {
			st = co.State.Terminated.Reason
		}
	}
	return st
}
