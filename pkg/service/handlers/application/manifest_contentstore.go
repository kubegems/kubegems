package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"
)

type rawObject struct {
	filename string
	content  []byte
	object   client.Object
}

type FsStore struct {
	scheme    *runtime.Scheme
	fs        billy.Filesystem
	resources map[string]*rawObject // filename -> resource
}

// NewGitFsStore 是一个代价比较高的操作
func NewGitFsStore(fs billy.Filesystem) *FsStore {
	contents := &FsStore{
		resources: map[string]*rawObject{},
		fs:        fs,
		scheme:    scheme.Scheme,
	}
	ForFileContentFunc(fs, "", func(filename string, content []byte) error {
		if filepath.Ext(filename) != ".yaml" {
			return nil
		}
		obj, _ := DecodeResource(content)
		// 如果解析不成则  obj 为 nil
		contents.resources[filename] = &rawObject{object: obj, content: content}
		return nil
	})
	return contents
}

func (c *FsStore) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if obj == nil || obj.GetName() == "" {
		return errors.NewBadRequest("empty name")
	}

	if filename, _ := c.find(obj); filename != "" {
		return errors.NewAlreadyExists(schema.GroupResource{}, obj.GetName())
	}

	filename := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind) + "-" + obj.GetName() + ".yaml"
	// create
	content, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	content = removeStatusField(content)
	if err := util.WriteFile(c.fs, filename, content, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func (c *FsStore) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if filename, _ := c.find(obj); filename != "" {
		_ = c.fs.Remove(filename)
		return nil
	}
	return errors.NewNotFound(schema.GroupResource{}, obj.GetName())
}

func (c *FsStore) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil // not implemented
}

func (c *FsStore) ListAll(ctx context.Context) ([]client.Object, error) {
	items := []client.Object{}
	for _, obj := range c.resources {
		if obj.object == nil {
			continue
		}
		items = append(items, obj.object)
	}
	return items, nil
}

func (c *FsStore) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	gvk, err := apiutil.GVKForObject(list, scheme.Scheme)
	if err != nil {
		return err
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	items := []client.Object{}
	for _, obj := range c.resources {
		if obj.object != nil && obj.object.GetObjectKind().GroupVersionKind() == gvk {
			items = append(items, obj.object)
		}
	}

	templist := &struct {
		Items []client.Object `json:"items"`
	}{
		Items: items,
	}

	listraw, err := yaml.Marshal(templist)
	if err != nil {
		return err
	}

	return runtime.DecodeInto(decoder, listraw, list)
}

func (c *FsStore) Get(ctx context.Context, key client.ObjectKey, into client.Object) error {
	into.SetName(key.Name)
	if filename, found := c.find(into); filename != "" {
		if err := runtime.DecodeInto(decoder, found.content, into); err != nil {
			return err
		}
		return nil
	}
	return errors.NewNotFound(schema.GroupResource{}, key.Name)
}

func (c *FsStore) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	options := client.PatchOptions{}
	options.ApplyOptions(opts)

	patchcontent, err := patch.Data(obj)
	if err != nil {
		return err
	}
	filename, found := c.find(obj)
	if filename == "" {
		return nil
	}

	var patchedjson []byte

	rawjson, err := yaml.YAMLToJSON(found.content)
	if err != nil {
		return err
	}
	// https://github.com/kubernetes/kubernetes/blob/release-1.20/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L337
	switch patch.Type() {
	case types.JSONPatchType:
		jsonpatchpatch, err := jsonpatch.DecodePatch(patchcontent)
		if err != nil {
			return err
		}
		patchedjson, err = jsonpatchpatch.Apply(rawjson)
		if err != nil {
			return err
		}
	case types.MergePatchType:
		patchedjson, err = jsonpatch.MergePatch(rawjson, patchcontent)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported patch type %s", patch.Type())
	}

	if err := json.Unmarshal(patchedjson, obj); err != nil {
		return err
	}
	patchedyaml, err := yaml.JSONToYAML(patchedjson)
	if err != nil {
		return err
	}

	patchedyaml = removeStatusField(patchedyaml)
	if err := util.WriteFile(c.fs, filename, patchedyaml, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func (c *FsStore) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if filename, _ := c.find(obj); filename != "" {
		// updated
		content, err := yaml.Marshal(obj)
		if err != nil {
			return err
		}
		content = removeStatusField(content)
		if err := util.WriteFile(c.fs, filename, content, os.ModePerm); err != nil {
			return err
		}
		return nil
	}
	return errors.NewNotFound(schema.GroupResource{}, obj.GetName())
}

// Scheme returns the scheme this client is using.
func (c *FsStore) Scheme() *runtime.Scheme {
	return c.scheme
}

// RESTMapper returns the rest this client is using.
func (c *FsStore) RESTMapper() meta.RESTMapper {
	return nil
}

func (c *FsStore) Status() client.StatusWriter {
	return c
}

func (c *FsStore) find(find client.Object) (string, *rawObject) {
	if gvk, err := apiutil.GVKForObject(find, scheme.Scheme); err != nil {
		return "", nil
	} else {
		find.GetObjectKind().SetGroupVersionKind(gvk)
	}

	for filename, item := range c.resources {
		if item.object == nil {
			continue // 这是非资源文件
		}
		if item.object.GetObjectKind().GroupVersionKind() != find.GetObjectKind().GroupVersionKind() {
			continue
		}
		if item.object.GetName() != find.GetName() {
			continue
		}
		return filename, item
	}
	return "", nil
}

func removeStatusField(origin []byte) []byte {
	tmp := map[string]interface{}{}
	if err := yaml.Unmarshal(origin, &tmp); err != nil {
		return origin
	}

	unstructured.RemoveNestedField(tmp, "status")
	if removed, err := yaml.Marshal(tmp); err != nil {
		return origin
	} else {
		return removed
	}
}
