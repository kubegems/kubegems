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

package utils

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"kubegems.io/kubegems/pkg/apis/plugins"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type DiffResult struct {
	Creats  []*unstructured.Unstructured
	Applys  []*unstructured.Unstructured
	Removes []*unstructured.Unstructured
}

func Diff(managed []corev1.ObjectReference, resources []*unstructured.Unstructured) DiffResult {
	result := DiffResult{}
	managedmap := map[corev1.ObjectReference]bool{}
	for _, item := range managed {
		managedmap[item] = false
	}
	for _, item := range resources {
		man := GetReference(item)
		if _, ok := managedmap[man]; !ok {
			result.Creats = append(result.Creats, item)
		} else {
			result.Applys = append(result.Applys, item)
		}
		managedmap[man] = true
	}
	for k, v := range managedmap {
		if !v {
			uns := &unstructured.Unstructured{}
			uns.SetAPIVersion(k.APIVersion)
			uns.SetKind(k.Kind)
			uns.SetName(k.Name)
			uns.SetNamespace(k.Namespace)
			result.Removes = append(result.Removes, uns)
		}
	}
	return result
}

func NewDefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		ServerSideApply: true,
		CreateNamespace: true,
	}
}

type SyncOptions struct {
	ServerSideApply bool
	CreateNamespace bool
}

type Apply struct {
	Client client.Client
}

func (a *Apply) Sync(ctx context.Context, managed []corev1.ObjectReference, resources []*unstructured.Unstructured, options *SyncOptions) ([]corev1.ObjectReference, error) {
	return a.SyncDiff(ctx, Diff(managed, resources), options)
}

func (a *Apply) SyncDiff(ctx context.Context, diff DiffResult, options *SyncOptions) ([]corev1.ObjectReference, error) {
	log := logr.FromContextOrDiscard(ctx)

	errs := []string{}

	managed := []corev1.ObjectReference{}
	// create
	for _, item := range diff.Creats {
		log.Info("creating resource", "resource", item.GetObjectKind().GroupVersionKind().String(), "name", item.GetName(), "namespace", item.GetNamespace())
		if options.CreateNamespace {
			a.createNsIfNotExists(ctx, item.GetNamespace())
		}
		if err := ApplyResource(ctx, a.Client, item, ApplyOptions{ServerSideApply: options.ServerSideApply}); err != nil {
			err = fmt.Errorf("%s %s/%s: %v", item.GetObjectKind().GroupVersionKind().String(), item.GetNamespace(), item.GetName(), err)
			log.Error(err, "creating resource")
			errs = append(errs, err.Error())
			continue
		}
		managed = append(managed, GetReference(item)) // set managed
	}

	// apply
	for _, item := range diff.Applys {
		managed = append(managed, GetReference(item)) // set managed

		if IsSkipedOn(item, plugins.AnnotationIgnoreOptionOnUpdate) {
			log.Info("ignoring update", "resource", item.GetObjectKind().GroupVersionKind().String(), "name", item.GetName(), "namespace", item.GetNamespace())
			continue
		}

		log.Info("applying resource", "resource", item.GetObjectKind().GroupVersionKind().String(), "name", item.GetName(), "namespace", item.GetNamespace())
		if options.CreateNamespace {
			a.createNsIfNotExists(ctx, item.GetNamespace())
		}
		if err := ApplyResource(ctx, a.Client, item, ApplyOptions{ServerSideApply: options.ServerSideApply}); err != nil {
			err = fmt.Errorf("%s %s/%s: %v", item.GetObjectKind().GroupVersionKind().String(), item.GetNamespace(), item.GetName(), err)
			log.Error(err, "applying resource")
			errs = append(errs, err.Error())
			continue
		}
	}
	// remove
	for _, item := range diff.Removes {

		if IsSkipedOn(item, plugins.AnnotationIgnoreOptionOnDelete) {
			log.Info("ignoring delete", "resource", item.GetObjectKind().GroupVersionKind().String(), "name", item.GetName(), "namespace", item.GetNamespace())
			continue
		}

		partial := item
		log.Info("deleting resource", "resource", partial.GetObjectKind().GroupVersionKind().String(), "name", partial.GetName(), "namespace", partial.GetNamespace())
		if err := a.Client.Delete(ctx, partial, &client.DeleteOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				err = fmt.Errorf("%s %s/%s: %v", partial.GetObjectKind().GroupVersionKind().String(), partial.GetNamespace(), partial.GetName(), err)
				log.Error(err, "deleting resource")
				errs = append(errs, err.Error())
				// if not removed, keep in managed
				managed = append(managed, GetReference(item)) // set managed
				continue
			}
		}
	}

	// sort manged
	sort.Slice(managed, func(i, j int) bool {
		return strings.Compare(managed[i].APIVersion, managed[j].APIVersion) == 1
	})
	if len(errs) > 0 {
		return managed, errors.New(strings.Join(errs, "\n"))
	} else {
		return managed, nil
	}
}

func (a *Apply) createNsIfNotExists(ctx context.Context, name string) error {
	if name == "" {
		return nil
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err := controllerutil.CreateOrUpdate(ctx, a.Client, ns, func() error { return nil })
	return err
}

type ApplyOptions struct {
	ServerSideApply bool
	FieldOwner      string
}

func ApplyResource(ctx context.Context, cli client.Client, obj client.Object, options ApplyOptions) error {
	if options.FieldOwner == "" {
		options.FieldOwner = "bundler"
	}

	exists, _ := obj.DeepCopyObject().(client.Object)
	if err := cli.Get(ctx, client.ObjectKeyFromObject(exists), exists); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return cli.Create(ctx, obj)
	}

	var patch client.Patch
	var patchoptions []client.PatchOption
	if options.ServerSideApply {
		obj.SetManagedFields(nil)
		patch = client.Apply
		patchoptions = append(patchoptions,
			client.FieldOwner(options.FieldOwner),
			client.ForceOwnership,
		)
	} else {
		patch = client.MergeFrom(exists)
	}

	// patch
	if err := cli.Patch(ctx, obj, patch, patchoptions...); err != nil {
		return err
	}
	return nil
}

func IsSkipedOn(obj client.Object, key string) bool {
	if annotations := obj.GetAnnotations(); annotations != nil {
		for _, opt := range strings.Split(annotations[plugins.AnnotationIgnoreOptions], ",") {
			if opt == key {
				return true
			}
		}
	}
	return false
}
