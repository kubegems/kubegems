package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-logr/logr"
	"github.com/kubegems/gems/pkg/utils/git"
	"github.com/opentracing/opentracing-go"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

const (
	ReadmeFilename = "readme.md"
	MetaFilename   = ".meta"
)

type ManifestProcessor struct {
	GitProvider *git.SimpleLocalProvider
}

func NewManifestProcessor(GitProvider *git.SimpleLocalProvider) (*ManifestProcessor, error) {
	return &ManifestProcessor{GitProvider: GitProvider}, nil
}

func (p *ManifestProcessor) ContentFunc(ctx context.Context, ref PathRef, fun RepositoryFileSystemFunc) error {
	return p.Func(ctx, ref,
		FsFunc(fun),
	)
}

func (h *ManifestProcessor) UpdateContentFunc(ctx context.Context, ref PathRef, fun RepositoryFileSystemFunc, commitmsg string) error {
	return h.Func(ctx, ref,
		Pull(), // git remote update 一次
		FsFunc(fun),
		UpdateKustomizeCommit(commitmsg),
	)
}

func (h *ManifestProcessor) StoreFunc(ctx context.Context, ref PathRef, fun func(ctx context.Context, store GitStore) error) error {
	return h.ContentFunc(ctx, ref, FSStoreFunc(fun))
}

func (h *ManifestProcessor) StoreUpdateFunc(ctx context.Context, ref PathRef, fun func(ctx context.Context, store GitStore) error, commitmsg string) error {
	return h.UpdateContentFunc(ctx, ref, FSStoreFunc(fun), commitmsg)
}

func (h *ManifestProcessor) Create(ctx context.Context, ref PathRef, manifest Manifest) error {
	updatefunc := func(_ context.Context, fs billy.Filesystem) error {
		// init sample data
		// 初始化文件必须在kustomize前
		if err := initSampleManifest(fs, manifest.Name, manifest.Kind); err != nil {
			return err
		}
		// init kustomization with labels
		if err := InitOrUpdateKustomization(fs, manifest.Labels, manifest.Annotations); err != nil {
			return err
		}
		// init readme
		if err := util.WriteFile(fs, ReadmeFilename, []byte(manifest.Description), os.ModePerm); err != nil {
			return err
		}
		// init creation meta
		if err := setManifestMeta(fs, manifestmeta{Creator: AuthorFromContext(ctx).Name, CreateAt: metav1.Now()}); err != nil {
			return err
		}
		return nil
	}
	if ref.Name == "" {
		ref.Name = manifest.Name
	}
	if err := h.UpdateContentFunc(ctx, ref, updatefunc, "init"); err != nil {
		return err
	}
	return nil
}

func initSampleManifest(fs billy.Filesystem, name, kind string) error {
	var sample interface{}

	labels := map[string]string{
		CommonLabelApplication: name,
		"app":                  name, // kubernetes app
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: "docker.io/library/hello-world:latest",
				},
			},
		},
	}
	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}
	metadata := metav1.ObjectMeta{Name: name}

	switch kind {
	case "Deployment":
		sample = &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "Deployment",
			},
			ObjectMeta: metadata,
			Spec:       appsv1.DeploymentSpec{Replicas: pointer.Int32(1), Selector: selector, Template: podTemplate},
		}
	case "DaemonSet":
		sample = &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "DaemonSet",
			},
			ObjectMeta: metadata,
			Spec:       appsv1.DaemonSetSpec{Selector: selector, Template: podTemplate},
		}
	case "StatefulSet":
		sample = &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "StatefulSet",
			},
			ObjectMeta: metadata,
			Spec:       appsv1.StatefulSetSpec{Replicas: pointer.Int32(1), Selector: selector, Template: podTemplate},
		}
	}

	bts, err := yaml.Marshal(sample)
	if err != nil {
		return err
	}
	return util.WriteFile(fs, strings.ToLower(kind)+".yaml", bts, os.ModePerm)
}

func (h *ManifestProcessor) Get(ctx context.Context, ref PathRef) (*Manifest, error) {
	manifest := &Manifest{Name: ref.Name, Ref: ref}
	h.ContentFunc(ctx, ref, func(ctx context.Context, fs billy.Filesystem) error {
		return DecodeManifest(ctx, fs, manifest, ManifestListOptions{WithImages: true})
	})
	return manifest, nil
}

type ManifestListOption func(*ManifestListOptions)

func WithImages() ManifestListOption {
	return func(o *ManifestListOptions) {
		o.WithImages = true
	}
}

func (h *ManifestProcessor) List(ctx context.Context, ref PathRef, options ...ManifestListOption) ([]Manifest, error) {
	opts := &ManifestListOptions{}
	for _, o := range options {
		o(opts)
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "list manifests")
	defer span.Finish()

	if ref.Name != "" {
		return nil, fmt.Errorf("name must be empty")
	}

	manifestList := []Manifest{}
	contentfunc := func(_ context.Context, fs billy.Filesystem) error {
		ForManifestFsFunc(fs, func(name string, ifs billy.Filesystem) {
			// in app
			manifest := Manifest{
				Name: name,
				Ref:  PathRef{Tenant: ref.Tenant, Project: ref.Project, Env: ref.Env, Name: name},
			}
			_ = DecodeManifest(ctx, ifs, &manifest, *opts)
			manifestList = append(manifestList, manifest)
		})
		return nil
	}

	if err := h.Func(ctx, ref,
		FsFunc(contentfunc),
		func(ctx context.Context, repository Repository) error {
			// 一次性填充创建时间
			decodeCreationTimestamp(ctx, repository, manifestList)
			return nil
		},
	); err != nil {
		return nil, err
	}

	sort.Slice(manifestList, func(i, j int) bool {
		return manifestList[i].CreateAt.After(manifestList[j].CreateAt.Time)
	})
	return manifestList, nil
}

func (h *ManifestProcessor) Update(ctx context.Context, ref PathRef, manifest Manifest) error {
	updatefunc := func(_ context.Context, fs billy.Filesystem) error {
		if err := util.WriteFile(fs, ReadmeFilename, []byte(manifest.Description), os.ModePerm); err != nil {
			return err
		}
		// update kustomization
		return InitOrUpdateKustomization(fs, manifest.Labels, manifest.Annotations)
	}
	return h.UpdateContentFunc(ctx, ref, updatefunc, "update description")
}

func (h *ManifestProcessor) Remove(ctx context.Context, ref PathRef) error {
	updatefunc := func(_ context.Context, fs billy.Filesystem) error {
		_ = util.RemoveAll(fs, ".")
		return nil
	}
	return h.UpdateContentFunc(ctx, ref, updatefunc, "remove")
}

func ForManifestFsFunc(fs billy.Filesystem, fun func(string, billy.Filesystem)) {
	fis, err := fs.ReadDir(".")
	if err != nil {
		return
	}
	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		// 空文件夹跳过
		if infos, _ := fs.ReadDir(fi.Name()); len(infos) == 0 {
			continue
		}
		name := fi.Name()
		fun(name, chroot.New(fs, name))
	}
}

type ManifestListOptions struct {
	WithImages bool // 解析资源中的镜像 和 istioVersion 更准确的kind等
}

var detectKinds = []string{
	"deployment",
	"statefulset",
	"daemonset",
	"cronjob",
	"job",
}

type manifestmeta struct {
	Creator  string      `json:"creator"`
	CreateAt metav1.Time `json:"createAt"`
}

// 从应用编排根目录获取编排详情
func DecodeManifest(ctx context.Context, fs billy.Filesystem, manifest *Manifest, options ManifestListOptions) error {
	if manifest.Name == "" {
		return errors.New("name cannot be empty")
	}

	// description
	desccontent, _ := util.ReadFile(fs, ReadmeFilename)
	manifest.Description = string(desccontent)

	// creation
	manifestmeta := &manifestmeta{}
	creationcontent, _ := util.ReadFile(fs, MetaFilename)
	_ = json.Unmarshal(creationcontent, manifestmeta)
	if !manifestmeta.CreateAt.IsZero() {
		manifest.CreateAt = manifestmeta.CreateAt
		manifest.Creator = manifestmeta.Creator
	}

	// kustomization
	kustomization := &types.Kustomization{}
	kustomizationcontent, _ := util.ReadFile(fs, KustimizationFilename)
	_ = yaml.Unmarshal(kustomizationcontent, kustomization)
	manifest.Annotations = kustomization.CommonAnnotations
	manifest.Labels = kustomization.CommonLabels

	// images version kind
	if options.WithImages {
		images := []string{}
		istioVersion := ""

		workload, _ := ParseMainWorkload(ctx, NewGitFsStore(fs))
		ObjectPodTemplateFunc(workload, func(template *corev1.PodTemplateSpec) {
			for _, c := range template.Spec.Containers {
				if v, ok := template.Labels[LabelIstioVersion]; ok {
					istioVersion = v
				}
				images = append(images, c.Image)
			}
		})
		// update
		manifest.Images = images
		manifest.IstioVersion = istioVersion
		if workload != nil {
			manifest.Kind = workload.GetObjectKind().GroupVersionKind().Kind
		}
	} else {
		// kind
		for _, res := range kustomization.Resources {
			for _, kind := range detectKinds {
				// 因为约定，文件名中一般包含 deployment statefulset 等字样，如果存在这些则可以直接判断
				if strings.Contains(res, kind) {
					manifest.Kind = kind
					break
				}
			}
		}
	}
	return nil
}

// 该方法仅作为备用方式
func decodeCreationTimestamp(ctx context.Context, repo Repository, manifests []Manifest) {
	fs, err := repo.FS(ctx)
	if err != nil {
		return
	}

	update := false
	// 是否存在 没有创建时间的
	for i, manifest := range manifests {
		if manifest.CreateAt.IsZero() {
			meta := manifestmeta{}
			// 直到路径下的最晚提交，作为创建时间
			// 这种方式的开销太大
			repo.repo.HistoryFunc(ctx, manifest.Name, func(_ context.Context, commit git.Commit) error {
				meta.CreateAt = metav1.NewTime(commit.Author.When)
				meta.Creator = commit.Author.Name
				return nil
			})

			// set meta
			update = true
			manifests[i].CreateAt = meta.CreateAt
			manifests[i].Creator = meta.Creator

			setManifestMeta(chroot.New(fs, manifest.Name), meta)
		}
	}
	if update {
		// 后台更新
		logr.FromContextOrDiscard(ctx).Info("init creation meta", "repo", repo.repo.CloneURL())
		go Commit("init meta")(ctx, repo)
	}
}

func setManifestMeta(fs billy.Filesystem, meta manifestmeta) error {
	content, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return util.WriteFile(fs, MetaFilename, content, os.ModePerm)
}
