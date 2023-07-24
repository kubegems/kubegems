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

package kube

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FillGVK(object client.Object, scheme *runtime.Scheme) error {
	gvks, unversioned, err := scheme.ObjectKinds(object)
	if err != nil {
		return err
	}
	gvk := gvks[0]
	object.GetObjectKind().SetGroupVersionKind(gvk)
	_ = unversioned
	return nil
}

func NewListOf(gvk schema.GroupVersionKind, scheme *runtime.Scheme) client.ObjectList {
	if !strings.HasSuffix(gvk.Kind, "List") {
		gvk.Kind = gvk.Kind + "List"
	}
	// try decode using typed ObjectList first
	list, err := scheme.New(gvk)
	if err != nil {
		// fallback to unstructured.UnstructuredList
		list = &unstructured.UnstructuredList{}
	}
	objlist, ok := list.(client.ObjectList)
	if !ok {
		// fallback to unstructured.UnstructuredList
		objlist = &unstructured.UnstructuredList{}
	}
	objlist.GetObjectKind().SetGroupVersionKind(gvk)
	return objlist
}
