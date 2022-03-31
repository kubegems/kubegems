package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/yaml"
)

func KustomizeBuildPlugin(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	return KustomizeBuild(ctx, plugin.Path)
}

// build kustomization
func KustomizeBuild(ctx context.Context, dir string) ([]*unstructured.Unstructured, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	m, err := k.Run(filesys.MakeFsOnDisk(), dir)
	if err != nil {
		return nil, err
	}
	yml, err := m.AsYaml()
	if err != nil {
		return nil, err
	}

	res := []*unstructured.Unstructured{}
	items, err := kube.SplitYAML(yml)
	if err != nil {
		return nil, fmt.Errorf("parse content [%s]: %v", string(yml), err)
	}
	res = append(res, items...)

	return res, nil
}

type Release struct {
	Name      string
	Namespace string
	Version   string
}

const ValuesFile = "values.yaml"

func TemplatesBuildPlugin(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	defaultValus := map[string]interface{}{}
	if content, err := os.ReadFile(filepath.Join(plugin.Path, ValuesFile)); err == nil {
		yaml.Unmarshal(content, &defaultValus)
	}

	values := MergeMaps(defaultValus, plugin.Values)

	tplValues := struct {
		Values  map[string]interface{}
		Release Release
	}{
		Values: values,
		Release: Release{
			Name:      plugin.Name,
			Namespace: plugin.Namespace,
			Version:   plugin.Version,
		},
	}

	enableds := parseEnabled(values)
	return templatesBuild(ctx, plugin.Path, tplValues, enableds)
}

func parseEnabled(values map[string]interface{}) map[string]bool {
	boolkvs := map[string]bool{}
	expandToKV([]string{}, values, boolkvs)

	ret := map[string]bool{}
	for k, v := range boolkvs {
		keys := strings.Split(k, ".")
		if keys[len(keys)-1] == "enabled" {
			ret[strings.Join(keys[:len(keys)-1], "/")] = v
		}
	}
	return ret
}

func expandToKV[T any](keys []string, values map[string]interface{}, into map[string]T) {
	for k, v := range values {
		keys := append(keys, k)
		if v, ok := v.(map[string]interface{}); ok {
			expandToKV(keys, v, into)
			continue
		}
		if val, ok := v.(T); ok {
			into[strings.Join(keys, ".")] = val
		}
	}
}

// valus in b overrides in a
func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func templatesBuild(ctx context.Context, basepath string, tplValues interface{}, enableds map[string]bool) ([]*unstructured.Unstructured, error) {
	var res []*unstructured.Unstructured
	if err := filepath.WalkDir(basepath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(basepath, path)
		if err != nil {
			return err
		}
		if rel == ValuesFile {
			return nil
		}

		// check enabled
		p := strings.TrimSuffix(rel, filepath.Ext(rel))
		if enabled, exist := enableds[p]; exist && !enabled {
			return nil
		}

		if ext := strings.ToLower(filepath.Ext(info.Name())); ext != ".json" && ext != ".yml" && ext != ".yaml" {
			return nil
		}
		// check enabled

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		// template
		data, err = templates(info.Name(), data, tplValues)
		if err != nil {
			return err
		}
		items, err := kube.SplitYAML(data)
		if err != nil {
			return fmt.Errorf("parse content [%s]: %v", string(data), err)
		}
		res = append(res, items...)
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func templates(name string, content []byte, values interface{}) ([]byte, error) {
	template, err := template.
		New(name).
		Option("missingkey=error").
		Funcs(sprig.TxtFuncMap()).
		Parse(string(content))
	if err != nil {
		return nil, err
	}
	result := bytes.NewBuffer(nil)
	if err := template.Execute(result, values); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}
