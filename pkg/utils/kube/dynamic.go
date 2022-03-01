package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"kubegems.io/pkg/log"
)

func CreateByYamlOrJson(ctx context.Context, cfg *rest.Config, yamlOrJson []byte) error {
	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 2. Prepare the dynamic client
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	log.Debugf(string(yamlOrJson))
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(yamlOrJson), 1024)

	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				return nil
			}

			return errors.Wrap(err, "decode raw")
		}
		// 3. Decode YAML manifest into unstructured.Unstructured
		obj := &unstructured.Unstructured{}
		_, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, obj)
		if err != nil {
			return errors.Wrap(err, "get gvk")
		}

		// 4. Find GVR
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return errors.Wrap(err, "rest mapping")
		}

		// 5. Obtain REST interface for the GVR
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			// namespaced resources should specify the namespace
			dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			// for cluster-wide resources
			dr = dyn.Resource(mapping.Resource)
		}

		// 6. Marshal object into JSON
		data, err := json.Marshal(obj)
		if err != nil {
			return errors.Wrap(err, "json marshal")
		}

		forceApplyMangagedFields := true
		// 7. Create or Update the object with SSA
		//     types.ApplyPatchType indicates SSA.
		//     FieldManager specifies the field owner ID.
		_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "gems-server",
			Force:        &forceApplyMangagedFields, // 参考 https://kubernetes.io/zh/docs/reference/using-api/server-side-apply/#conflicts
		})

		if err != nil {
			log.Errorf("failed to apply %s %s, err: %v, yaml:\n%s", obj.GetKind(), obj.GetName(), err, string(data))
			return err
		}
		log.Info("apply succeed", "kind", obj.GetKind(), "name", obj.GetName())
	}
}

func DeleteByYamlOrJson(ctx context.Context, cfg *rest.Config, yamlOrJson []byte) error {
	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 2. Prepare the dynamic client
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	log.Debugf(string(yamlOrJson))
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(yamlOrJson), 1024)

	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Wrap(err, "decode raw")
		}
		// 3. Decode YAML manifest into unstructured.Unstructured
		obj := &unstructured.Unstructured{}
		_, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, obj)
		if err != nil {
			return errors.Wrap(err, "get gvk")
		}

		// 4. Find GVR
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return errors.Wrap(err, "rest mapping")
		}

		// 5. Obtain REST interface for the GVR
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			// namespaced resources should specify the namespace
			dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			// for cluster-wide resources
			dr = dyn.Resource(mapping.Resource)
		}

		// 6. Marshal object into JSON
		data, err := json.Marshal(obj)
		if err != nil {
			return errors.Wrap(err, "json marshal")
		}

		if err := dr.Delete(ctx, obj.GetName(), metav1.DeleteOptions{}); err != nil {
			log.Errorf("failed to delete %s %s, err: %v, yaml:\n%s", obj.GetKind(), obj.GetName(), err, string(data))
			return err
		}
		log.Info("delete succeed", "kind", obj.GetKind(), "name", obj.GetName())
	}
}
