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

package statistics

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func FilterResourceName(list corev1.ResourceList, keep func(name corev1.ResourceName) bool) corev1.ResourceList {
	ret := corev1.ResourceList{}
	for k, v := range list {
		if keep(k) {
			ret[k] = v.DeepCopy()
		}
	}
	return ret
}

func AppendResourceNamePrefix(prefix string, list corev1.ResourceList) corev1.ResourceList {
	ret := corev1.ResourceList{}
	for k, v := range list {
		ret[corev1.ResourceName(prefix)+k] = v.DeepCopy()
	}
	return ret
}

func AddResourceList(total corev1.ResourceList, add corev1.ResourceList) {
	ResourceListCollect(total, add, func(_ corev1.ResourceName, into *resource.Quantity, val resource.Quantity) {
		into.Add(val)
	})
}

func SubResourceList(total corev1.ResourceList, sub corev1.ResourceList) {
	ResourceListCollect(total, sub, func(_ corev1.ResourceName, into *resource.Quantity, val resource.Quantity) {
		into.Sub(val)
	})
}

type ResourceListCollectFunc func(corev1.ResourceName, *resource.Quantity, resource.Quantity)

func ResourceListCollect(into, vals corev1.ResourceList, collect ResourceListCollectFunc) corev1.ResourceList {
	for resourceName, quantity := range vals {
		lastQuantity := into[resourceName].DeepCopy()
		collect(resourceName, &lastQuantity, quantity)
		into[resourceName] = lastQuantity
	}
	return into
}
