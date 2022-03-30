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

func TemplatesBuildPlugin(ctx context.Context, plugin Plugin) ([]*unstructured.Unstructured, error) {
	tplValues := struct {
		Values  map[string]interface{}
		Release Release
	}{
		Values: plugin.Values,
		Release: Release{
			Name:      plugin.Name,
			Namespace: plugin.Namespace,
			Version:   plugin.Version,
		},
	}
	return TemplatesBuild(ctx, plugin.Path, tplValues)
}

func TemplatesBuild(ctx context.Context, path string, tplValues interface{}) ([]*unstructured.Unstructured, error) {
	var res []*unstructured.Unstructured
	if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(info.Name())); ext != ".json" && ext != ".yml" && ext != ".yaml" {
			return nil
		}
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
