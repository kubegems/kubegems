package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
)

type BuildFunc func(ctx context.Context, plugin *pluginsv1beta1.Plugin, dir string) ([]byte, error)

type NativeApplier struct {
	Client   client.Client
	Builders map[pluginsv1beta1.PluginKind]BuildFunc
}

func NewNative(cfg *rest.Config, cli client.Client) *NativeApplier {
	return &NativeApplier{
		Client: cli,
		Builders: map[pluginsv1beta1.PluginKind]BuildFunc{
			pluginsv1beta1.PluginKindKustomize: KustomizeBuild,
			pluginsv1beta1.PluginKindTemplate:  Templater{Config: cfg}.Template,
		},
	}
}

func KustomizeBuild(ctx context.Context, _ *pluginsv1beta1.Plugin, dir string) ([]byte, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	m, err := k.Run(filesys.MakeFsOnDisk(), dir)
	if err != nil {
		return nil, err
	}
	yml, err := m.AsYaml()
	if err != nil {
		return nil, err
	}
	return []byte(yml), nil
}

func (p *NativeApplier) Template(ctx context.Context, bundle *pluginsv1beta1.Plugin, dir string) ([]byte, error) {
	builder, ok := p.Builders[bundle.Spec.Kind]
	if !ok {
		return nil, fmt.Errorf("no builder for kind %s", bundle.Spec.Kind)
	}
	return builder(ctx, bundle, dir)
}

func (p *NativeApplier) Apply(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) error {
	// check values
	log := logr.FromContextOrDiscard(ctx)
	if bundle.Status.Phase == pluginsv1beta1.PluginPhaseInstalled && reflect.DeepEqual(bundle.Spec.Values.Object, bundle.Status.Values.Object) {
		log.Info("values are up to date")
		return nil
	}
	log.Info("sync native plugin", "values", bundle.Spec.Values.Object)

	render, err := p.Template(ctx, bundle, into)
	if err != nil {
		return err
	}
	resources, err := SplitYAML(render)
	if err != nil {
		return err
	}

	ns := bundle.Spec.InstallNamespace
	if ns != "" {
		ns = bundle.Namespace
	}
	// override namespace
	setNamespaceIfNotSet(ns, resources)

	managedResources, err := p.Sync(ctx, resources, bundle.Status.Managed, &SyncOptions{ServerSideApply: true, CreateNamespace: true})
	if err != nil {
		return err
	}
	bundle.Status.Values = pluginsv1beta1.Values{Object: bundle.Spec.Values.Object}
	bundle.Status.Managed = managedResources
	bundle.Status.InstallNamespace = ns
	bundle.Status.Version = bundle.Spec.Version
	bundle.Status.Phase = pluginsv1beta1.PluginPhaseInstalled
	now := metav1.Now()
	bundle.Status.UpgradeTimestamp = now
	if bundle.Status.CreationTimestamp.IsZero() {
		bundle.Status.CreationTimestamp = now
	}
	bundle.Status.Message = ""
	return nil
}

func (p *NativeApplier) Remove(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("removing plugin")

	managedResources, err := p.Sync(ctx, nil, bundle.Status.Managed, &SyncOptions{
		ServerSideApply: true,
		CreateNamespace: true,
	})
	if err != nil {
		return err
	}
	bundle.Status.Managed = managedResources
	bundle.Status.Phase = pluginsv1beta1.PluginPhaseNone
	bundle.Status.Message = ""
	now := metav1.Now()
	bundle.Status.DeletionTimestamp = &now
	return nil
}

func setNamespaceIfNotSet(ns string, list []*unstructured.Unstructured) {
	for _, item := range list {
		if item.GetNamespace() == "" {
			item.SetNamespace(ns)
		}
	}
}

type SyncOptions struct {
	ServerSideApply bool
	CreateNamespace bool
}

func (a *NativeApplier) Sync(
	ctx context.Context,
	resources []*unstructured.Unstructured,
	premanaged []pluginsv1beta1.ManagedResource,
	options *SyncOptions,
) ([]pluginsv1beta1.ManagedResource, error) {
	log := logr.FromContextOrDiscard(ctx)
	managedmap := map[pluginsv1beta1.ManagedResource]bool{}
	for _, item := range premanaged {
		managedmap[item] = false
	}
	// apply
	errs := []string{}
	for _, item := range resources {
		log.Info("applying resource", "resource", item.GetObjectKind().GroupVersionKind().String(), "name", item.GetName(), "namespace", item.GetNamespace())
		if options.CreateNamespace {
			a.createNsIfNotExists(ctx, item.GetNamespace())
		}
		if err := a.apply(ctx, item, options.ServerSideApply); err != nil {
			err = fmt.Errorf("%s %s/%s: %v", item.GetObjectKind().GroupVersionKind().String(), item.GetNamespace(), item.GetName(), err)
			log.Error(err, "applying resource")
			errs = append(errs, err.Error())
			continue
		}
		managedmap[manFromResource(item)] = true // set managed
	}
	// remove
	for item, managed := range managedmap {
		if managed {
			continue
		}
		partial := partialFromMan(item)
		log.Info("deleting resource", "gvk", partial.GetObjectKind().GroupVersionKind().String(), "name", partial.GetName(), "namespace", partial.GetNamespace())
		if err := a.Client.Delete(ctx, partial, &client.DeleteOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				err = fmt.Errorf("%s %s/%s: %v", partial.GetObjectKind().GroupVersionKind().String(), partial.GetNamespace(), partial.GetName(), err)
				log.Error(err, "deleting resource")
				errs = append(errs, err.Error())
				// if not removed, keep in managed
				continue
			}
		}
		// remove from managed if removed
		delete(managedmap, manFromResource(partial))
	}
	managedResources := make([]pluginsv1beta1.ManagedResource, 0, len(resources))
	for item := range managedmap {
		managedResources = append(managedResources, item)
	}
	sort.Slice(managedResources, func(i, j int) bool {
		return strings.Compare(managedResources[i].APIVersion, managedResources[j].APIVersion) == 1
	})
	if len(errs) > 0 {
		return managedResources, errors.New(strings.Join(errs, "\n"))
	} else {
		return managedResources, nil
	}
}

func manFromResource(obj client.Object) pluginsv1beta1.ManagedResource {
	return pluginsv1beta1.ManagedResource{
		APIVersion: obj.GetObjectKind().GroupVersionKind().GroupVersion().String(),
		Kind:       obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace:  obj.GetNamespace(),
		Name:       obj.GetName(),
	}
}

func partialFromMan(man pluginsv1beta1.ManagedResource) *metav1.PartialObjectMetadata {
	return &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: man.APIVersion,
			Kind:       man.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      man.Name,
			Namespace: man.Namespace,
		},
	}
}

func (a *NativeApplier) createNsIfNotExists(ctx context.Context, name string) error {
	if name == "" {
		return nil
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err := controllerutil.CreateOrUpdate(ctx, a.Client, ns, func() error { return nil })
	return err
}

func (a *NativeApplier) apply(ctx context.Context, obj client.Object, serversideapply bool) error {
	key := client.ObjectKeyFromObject(obj)
	if err := a.Client.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		// create
		if err := a.Client.Create(ctx, obj); err != nil {
			return err
		}
		return nil
	}

	var patch client.Patch
	var patchoptions []client.PatchOption
	if serversideapply {
		obj.SetManagedFields(nil)
		patch = client.Apply
		patchoptions = append(patchoptions,
			client.FieldOwner("bundler"),
			client.ForceOwnership,
		)
	} else {
		patch = client.MergeFrom(obj)
	}

	// patch
	if err := a.Client.Patch(ctx, obj, patch, patchoptions...); err != nil {
		return err
	}
	return nil
}

func diff[T client.Object](a []T, b []pluginsv1beta1.ManagedResource) []pluginsv1beta1.ManagedResource {
	for _, item := range a {
		if i := indexOf(item, b); i >= 0 {
			b = append(b[:i], b[i+1:]...)
		}
	}
	return b
}

func indexOf(item client.Object, list []pluginsv1beta1.ManagedResource) int {
	for i, l := range list {
		if l.APIVersion == item.GetObjectKind().GroupVersionKind().GroupVersion().Identifier() &&
			l.Kind == item.GetObjectKind().GroupVersionKind().Kind &&
			l.Namespace == item.GetNamespace() &&
			l.Name == item.GetName() {
			return i
		}
	}
	return -1
}

func SplitYAML(data []byte) ([]*unstructured.Unstructured, error) {
	const cachesize = 4096
	d := kubeyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), cachesize)
	var objs []*unstructured.Unstructured
	for {
		u := &unstructured.Unstructured{}
		if err := d.Decode(u); err != nil {
			if err == io.EOF {
				break
			}
			return objs, fmt.Errorf("failed to unmarshal manifest: %v", err)
		}
		if len(u.Object) == 0 {
			continue
		}
		objs = append(objs, u)
	}
	return objs, nil
}
