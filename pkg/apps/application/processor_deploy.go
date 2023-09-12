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

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-logr/logr"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	istioclinetworkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/workflow"
	"sigs.k8s.io/yaml"
)

const TaskGroupApplication = "application"

const (
	TaskFunction_Application_UpdateImages              = "application_update_images"
	TaskFunction_Application_Sync                      = "application_sync"
	TaskFunction_Application_BatchUpdateImages         = "application_batch_update_images"
	TaskFunction_Application_PrepareDeploymentStrategy = "application_preparedeploymentstrategy"
	TaskFunction_Application_WaitRollouts              = "application_wait_rollouts"
	TaskFunction_Application_Undo                      = "application_undo"
)

// ProvideFuntions 用于对异步任务框架指出所使用的方法
func (p *ApplicationProcessor) ProvideFuntions() map[string]interface{} {
	return map[string]interface{}{
		TaskFunction_Application_UpdateImages:              p.UpdateImages,
		TaskFunction_Application_Sync:                      p.Sync,
		TaskFunction_Application_BatchUpdateImages:         p.BatchUpdateImages,
		TaskFunction_Application_PrepareDeploymentStrategy: p.PrepareDeploymentStrategyWithImages,
		TaskFunction_Application_WaitRollouts:              p.WaitRollouts,
		TaskFunction_Application_Undo:                      p.Undo,
	}
}

type ApplicationProcessor struct {
	Agents   *agents.ClientSet
	Argo     *argo.Client
	DataBase *DatabseProcessor
	Manifest *ManifestProcessor
	Task     *TaskProcessor

	// 缓存已经创建的 cluster,project,repo
	argostatuscache *sync.Map
}

func NewApplicationProcessor(db *database.Database, gitp git.Provider, argo *argo.Client, redis *redis.Client, agents *agents.ClientSet) *ApplicationProcessor {
	p := &ApplicationProcessor{
		Agents:   agents,
		Argo:     argo,
		DataBase: &DatabseProcessor{DB: db.DB()},
		Manifest: &ManifestProcessor{GitProvider: gitp},
		Task:     &TaskProcessor{Workflowcli: workflow.NewClientFromRedisClient(redis.Client)},

		argostatuscache: &sync.Map{},
	}
	return p
}

func (p *ApplicationProcessor) UpdateImages(ctx context.Context, ref PathRef, images []string, version string) error {
	updatefunc := func(ctx context.Context, store GitStore) error {
		return UpdateContentImages(ctx, store, images, version)
	}
	return p.Manifest.StoreUpdateFunc(ctx, ref, updatefunc, fmt.Sprintf("set images[%s],version[%s]", images, version))
}

type AppStoreDeployForm struct {
	Name         string            `json:"name" binding:"required"`
	Annotations  map[string]string `json:"annotations"`
	Labels       map[string]string `json:"labels"`
	RepoURL      string            `json:"repoURL" binding:"required"`      // 仓库index地址
	Chart        string            `json:"chart" binding:"required"`        // chart名称
	ChartVersion string            `json:"chartVersion" binding:"required"` // chart版本
	Values       json.RawMessage   `json:"values" binding:"required"`
}

func (h *ApplicationProcessor) Get(ctx context.Context, ref PathRef) (*DeploiedManifest, error) {
	manifest, err := h.Manifest.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	dm := &DeploiedManifest{Manifest: *manifest}
	app, _ := h.Argo.GetArgoApp(ctx, ref.FullName())
	return CompleteDeploiedManifestRuntime(app, dm), nil
}

func (h *ApplicationProcessor) List(ctx context.Context, ref PathRef) ([]DeploiedManifest, error) {
	manifests, err := h.Manifest.List(ctx, ref, WithImages())
	if err != nil {
		return nil, err
	}

	// list argo apps
	selector := labels.Set{
		LabelKeyFrom:     LabelValueFromApp,
		LabelTenant:      ref.Tenant,
		LabelProject:     ref.Project,
		LabelEnvironment: ref.Env,
	}.AsSelector()
	applist, err := h.Argo.ListArgoApp(ctx, selector)
	if err != nil {
		return nil, err
	}

	statusmap := map[string]*DeploiedManifest{}
	for _, manifest := range manifests {
		statusmap[manifest.Name] = &DeploiedManifest{
			Manifest: manifest,
			Runtime: ManifestRuntime{
				Status:  StatusNoArgoApp,
				Message: "no argo application found",
			},
		}
	}

	// argo 覆盖git
	for i := range applist.Items {
		app := applist.Items[i]
		appname := app.Labels[LabelApplication]
		if appname == "" {
			continue
		}
		if deploied, ok := statusmap[appname]; ok {
			statusmap[appname] = CompleteDeploiedManifestRuntime(&app, deploied)
		} else {
			statusmap[appname] = CompleteDeploiedManifestRuntime(&app, &DeploiedManifest{})
		}
	}
	// tolist
	deploiedManifests := make([]DeploiedManifest, 0, len(statusmap))
	for _, item := range statusmap {
		deploiedManifests = append(deploiedManifests, *item)
	}

	// sort
	sort.SliceStable(deploiedManifests, func(i, j int) bool {
		ir, jr := deploiedManifests[i].Runtime.CreateAt, deploiedManifests[j].Runtime.CreateAt
		if ir.IsZero() {
			ir = deploiedManifests[i].CreateAt
		}
		if jr.IsZero() {
			jr = deploiedManifests[j].CreateAt
		}
		if !ir.Equal(&jr) {
			return ir.After(jr.Time)
		}
		return strings.Compare(deploiedManifests[i].Name, deploiedManifests[j].Name) < 1
	})

	// sync database
	if err := h.DataBase.SyncDeploies(ctx, ref, deploiedManifests); err != nil {
		log.Error(err, "sync database failed")
	}

	return deploiedManifests, nil
}

func CompleteDeploiedManifestRuntime(app *v1alpha1.Application, status *DeploiedManifest) *DeploiedManifest {
	if status.Ref.IsEmpty() {
		status.Ref.FromArgoLabel(app.Labels)
	}
	if app == nil || app.CreationTimestamp.IsZero() {
		status.Runtime.Status = StatusNoArgoApp
		return status
	}
	status.Runtime.Labels = app.Labels
	status.Runtime.Annotations = app.Annotations
	if creator, ok := app.Annotations[AnnotationKeyCreator]; ok {
		status.Runtime.Creator = creator

		if status.Creator == "" {
			status.Creator = creator
		}
	}
	// 当编排不存在时从runtime回填
	if name, ok := app.Labels[LabelApplication]; status.Name == "" && ok {
		status.Name = name
	}
	status.Runtime.CreateAt = app.CreationTimestamp
	if status.CreateAt.IsZero() {
		status.CreateAt = app.CreationTimestamp
	}

	status.Runtime.Images = app.Status.Summary.Images
	if status.Images == nil {
		status.Images = app.Status.Summary.Images
	}

	status.Runtime.Status = string(app.Status.Health.Status)
	status.Runtime.Message = app.Status.Health.Message

	mainworkload := getMainManagerResource(*app)
	status.Runtime.Kind = mainworkload.Kind
	status.Runtime.WorkloadName = mainworkload.Name
	if status.Runtime.Kind != "" && status.Kind == "" {
		status.Kind = status.Runtime.Kind
	}
	status.Runtime.Raw = app
	return status
}

func (h *ApplicationProcessor) CreateBatch(ctx context.Context, baseref PathRef, names []string) error {
	var srcfs billy.Filesystem
	srcref := PathRef{Tenant: baseref.Tenant, Project: baseref.Project, Env: ""}
	err := h.Manifest.ContentFunc(ctx, srcref, func(ctx context.Context, fs billy.Filesystem) error {
		srcfs = fs
		return nil
	})
	if err != nil {
		return err
	}

	copyfilefunc := func(ctx context.Context, fs billy.Filesystem) error {
		for _, name := range names {
			err := ForFileContentFunc(srcfs, name, func(filename string, content []byte) error {
				destfile := filepath.Join(name, filename)
				return util.WriteFile(fs, destfile, content, os.ModePerm)
			})
			if err != nil {
				return err
			}
		}
		return nil
	}

	return h.Manifest.Func(ctx, baseref,
		Pull(),
		FsFunc(copyfilefunc),
		Commit("batch create"),
	)
}

func (h *ApplicationProcessor) Create(ctx context.Context, ref PathRef) error {
	// 先git copy
	files := []FileContent{}

	manifest, err := h.Manifest.Get(ctx, ref)
	if err != nil {
		return err
	}
	// copy from src
	srcref := PathRef{Tenant: ref.Tenant, Project: ref.Project, Env: "", Name: ref.Name} // base env
	if err := h.Manifest.ContentFunc(ctx, srcref, func(ctx context.Context, fs billy.Filesystem) error {
		return ForFileContentFunc(fs, "", func(filename string, content []byte) error {
			files = append(files, FileContent{Name: filename, Content: string(content)})
			return nil
		})
	}); err != nil {
		return err
	}
	// write to dest
	writetoenvmanifestfunc := func(_ context.Context, fs billy.Filesystem) error {
		if err := util.RemoveAll(fs, "."); err != nil {
			return err
		}
		// copy files
		for _, f := range files {
			if err := util.WriteFile(fs, f.Name, []byte(f.Content), os.ModePerm); err != nil {
				return err
			}
		}
		// set meta
		if err := setManifestMeta(fs, manifestmeta{Creator: AuthorFromContext(ctx).Name, CreateAt: metav1.Now()}); err != nil {
			return err
		}
		return nil
	}
	if err := h.Manifest.UpdateContentFunc(ctx, ref, writetoenvmanifestfunc, "manifest from base"); err != nil {
		return err
	}

	// deploy argo app
	if _, err := h.deployKustomizeApplication(ctx, ref, false); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}

	// sync database
	if err := h.DataBase.SyncDeploy(ctx, ref, DeploiedManifest{Manifest: *manifest}); err != nil {
		log.Error(err, "sync database failed")
	}

	return nil
}

func (h *ApplicationProcessor) deployHelmApplication(ctx context.Context, ref PathRef, form AppStoreDeployForm) (*v1alpha1.Application, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "create helm app")
	defer span.Finish()

	// 相比紧凑的json yaml 更人类可读
	yamlValues, err := yaml.JSONToYAML(form.Values)
	if err != nil {
		return nil, err
	}

	sethelmfunc := func(app *v1alpha1.Application) error {
		app.Labels[LabelKeyFrom] = LabelValueFromAppStore // is from app
		app.Annotations = labels.Merge(form.Annotations, app.Annotations)
		app.Labels = labels.Merge(form.Labels, app.Labels)
		app.Spec.Source = v1alpha1.ApplicationSource{
			RepoURL:        form.RepoURL,
			TargetRevision: form.ChartVersion,
			Chart:          form.Chart,
			Helm: &v1alpha1.ApplicationSourceHelm{
				ReleaseName: form.Name, // argo appname will be short when too long
				Values:      string(yamlValues),
			},
		}
		app.Operation = &v1alpha1.Operation{
			InitiatedBy: v1alpha1.OperationInitiator{
				Username: AuthorFromContext(ctx).Name,
			},
			Sync: &v1alpha1.SyncOperation{},
		}
		return nil
	}

	return h.deployArgoApplication(ctx, ref, sethelmfunc)
}

// https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#declarative-setup
func (h *ApplicationProcessor) deployKustomizeApplication(ctx context.Context, ref PathRef, sync bool) (*v1alpha1.Application, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "create kustomize app")
	defer span.Finish()

	var cloneurl string
	err := h.Manifest.Func(ctx, ref, func(_ context.Context, repository Repository) error {
		cloneurl = repository.repo.CloneURL()
		return nil
	})
	if err != nil {
		return nil, err
	}

	opts := h.Manifest.GitProvider.Options()

	// 目前只有kustomize用到 git repo
	if _, err := h.createArgoGitRepo(ctx, cloneurl, opts.Username, opts.Password); err != nil {
		return nil, fmt.Errorf("create argo repo: %w", err)
	}
	setkustomizefunc := func(app *v1alpha1.Application) error {
		app.Labels[LabelKeyFrom] = LabelValueFromApp // is from app
		// https://argoproj.github.io/argo-rollouts/features/traffic-management/istio/#integrating-with-gitops
		app.Spec.IgnoreDifferences = []v1alpha1.ResourceIgnoreDifferences{
			{
				Group:        istioclinetworkingv1alpha3.SchemeGroupVersion.Group,
				Kind:         "VirtualService",
				JSONPointers: []string{"/spec/http/0"}, // 忽略 argo rollouts
			},
			{
				Group: appsv1.SchemeGroupVersion.Group,
				Kind:  "Deployment",
				JSONPointers: []string{
					"/spec/template/metadata/annotations/sidecar.istio.io~1inject",         // 忽略istio annotation
					"/spec/template/metadata/annotations/sidecar.jaegertracing.io~1inject", // 忽略 jaeger annotation
				},
			},
			{
				Group: appsv1.SchemeGroupVersion.Group,
				Kind:  "StatefulSet",
				JSONPointers: []string{
					"/spec/template/metadata/annotations/sidecar.istio.io~1inject",         // 忽略istio annotation
					"/spec/template/metadata/annotations/sidecar.jaegertracing.io~1inject", // 忽略 jaeger annotation
				},
			},
		}
		app.Spec.SyncPolicy = &v1alpha1.SyncPolicy{
			SyncOptions: v1alpha1.SyncOptions{"ApplyOutOfSyncOnly=true"},
		}

		app.Spec.Source = v1alpha1.ApplicationSource{
			RepoURL:        cloneurl,
			Path:           ref.Path(),
			TargetRevision: ref.GitBranch(),
			Kustomize: &v1alpha1.ApplicationSourceKustomize{
				CommonAnnotations: map[string]string{
					AnnotationRef: string(ref.JsonStringBase64()),
				},
				// kustomize.yaml 中设置了 label，不需要再设置label了，避免对编排做出额外改动
				CommonLabels: map[string]string{},
				Images:       v1alpha1.KustomizeImages{}, // 设置为空
			},
		}
		if sync {
			app.Operation = &v1alpha1.Operation{
				InitiatedBy: v1alpha1.OperationInitiator{
					Username: AuthorFromContext(ctx).Name,
				},
				Sync: &v1alpha1.SyncOperation{
					Revision:    app.Spec.Source.TargetRevision,
					SyncOptions: app.Spec.SyncPolicy.SyncOptions,
				},
			}
		}
		return nil
	}
	return h.deployArgoApplication(ctx, ref, setkustomizefunc)
}

func GenArgoPrjNameFromRef(ref PathRef) string {
	return ref.Tenant + "-" + ref.Project + "-" + ref.Env
}

func GenArgoClusterName(cluster string) string {
	return "argocd-cluster-" + cluster
}

func DecodeArgoClusterName(full string) string {
	return strings.TrimPrefix(full, "argocd-cluster-")
}

// 从argo管理的资源中选出一个workload 类型作为argo类型
func getMainManagerResource(argoapp v1alpha1.Application) v1alpha1.ResourceStatus {
	ret := v1alpha1.ResourceStatus{}
	priority := -1
	for _, v := range argoapp.Status.Resources {
		if p, ok := kindPriorityMap[v.Kind]; ok && p > priority {
			ret = v
			priority = p
		}
	}
	return ret
}

var kindPriorityMap = map[string]int{
	"Replicaset":  0,
	"Job":         1,
	"CronJob":     2,
	"Deployment":  3,
	"StatefulSet": 4,
	"DaemonSet":   5,
}

// https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters
func (h *ApplicationProcessor) createArgoCluster(ctx context.Context, clustername string, kubeconfig []byte) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ensure argo cluster")
	defer span.Finish()

	var cluster *v1alpha1.Cluster
	if len(kubeconfig) == 0 {
		// use in cluster config
		cluster = &v1alpha1.Cluster{
			Name:   GenArgoClusterName(clustername),
			Server: "https://kubernetes.default.svc",
			Config: v1alpha1.ClusterConfig{
				TLSClientConfig: v1alpha1.TLSClientConfig{Insecure: true},
			},
		}
	} else {
		apiserver, cert, key, ca, err := kube.GetKubeconfigInfos(kubeconfig)
		if err != nil {
			return "", err
		}
		cluster = &v1alpha1.Cluster{
			Name:   GenArgoClusterName(clustername),
			Server: apiserver,
			Config: v1alpha1.ClusterConfig{
				TLSClientConfig: v1alpha1.TLSClientConfig{
					CertData: cert,
					KeyData:  key,
					CAData:   ca,
				},
			},
		}
	}

	cachekey := "cluster/" + cluster.Name
	if val, ok := h.argostatuscache.Load(cachekey); ok {
		// check kubeconfig
		exists := val.(*v1alpha1.Cluster)
		if string(exists.Config.CertData) == string(cluster.Config.CertData) {
			return exists.Name, nil
		}
		logr.FromContextOrDiscard(ctx).Info("cluster config changed", "cluster", clustername)
	}

	existcluster, err := h.Argo.EnsureCluster(ctx, cluster)
	if err != nil {
		return "", err
	}

	// cache it
	h.argostatuscache.Store(cachekey, existcluster)
	return existcluster.Name, nil
}

func (h *ApplicationProcessor) createArgoProjectForEnvironment(
	ctx context.Context, ref PathRef, clusterName string, namespace string,
) (*v1alpha1.AppProject, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ensure argo project")
	defer span.Finish()

	argoprj := &v1alpha1.AppProject{
		TypeMeta: v1.TypeMeta{
			Kind:       application.AppProjectKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: GenArgoPrjNameFromRef(ref),
		},
		// 目前暂时先设置允许所有目的环境，因为argo是私有的可放开限制
		// 若需要根据环境进行限制则需要根据 project environment 变化来更新
		Spec: v1alpha1.AppProjectSpec{
			SourceRepos: []string{"*"},
			Destinations: []v1alpha1.ApplicationDestination{
				{
					Name:      clusterName,
					Namespace: namespace,
				},
			},
			// 暂时不允许全局资源创建
			// ClusterResourceWhitelist: []v1.GroupKind{
			// 	{
			// 		Group: "*",
			// 		Kind:  "*",
			// 	},
			// },
		},
	}

	cachekey := "project/" + argoprj.Name
	if val, ok := h.argostatuscache.Load(cachekey); ok {
		return val.(*v1alpha1.AppProject), nil
	}

	existproject, err := h.Argo.EnsureArgoProject(ctx, argoprj)
	if err != nil {
		return nil, err
	}

	// cache it
	h.argostatuscache.Store(cachekey, existproject)
	return existproject, nil
}

// https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#repositories
func (h *ApplicationProcessor) createArgoGitRepo(ctx context.Context, gitCloneUrl string, username, password string) (*v1alpha1.Repository, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ensure argo repo")
	defer span.Finish()

	repo := &v1alpha1.Repository{
		Repo:     gitCloneUrl,
		Username: username,
		Password: password,
	}

	cachekey := "repository/" + repo.Repo
	if val, ok := h.argostatuscache.Load(cachekey); ok {
		return val.(*v1alpha1.Repository), nil
	}

	existrepo, err := h.Argo.EnsureRepository(ctx, repo)
	if err != nil {
		return nil, err
	}

	// cache it
	h.argostatuscache.Store(cachekey, existrepo)
	return existrepo, nil
}

func (h *ApplicationProcessor) Sync(ctx context.Context, ref PathRef, resources ...v1alpha1.SyncOperationResource) error {
	// do check in case of cluster config update
	if _, _, _, err := h.prepareArgoApplication(ctx, ref); err != nil {
		return err
	}
	if err := h.Argo.Sync(ctx, ref.FullName(), resources); err != nil {
		if !errors.IsNotFound(err) && grpcstatus.Code(err) != grpccodes.NotFound {
			return fmt.Errorf("sync app %s: %v", ref.Name, err)
		}
		// if not found do a fully deploy
		if _, err := h.deployKustomizeApplication(ctx, ref, true); err != nil {
			return fmt.Errorf("deploy app %s: %w", ref.Name, err)
		}
		return nil
	}
	return nil
}

func (h *ApplicationProcessor) prepareArgoApplication(ctx context.Context, ref PathRef) (clustername string, projectname string, namespace string, err error) {
	envdetails, err := h.DataBase.GetEnvironmentWithCluster(ref)
	if err != nil {
		return "", "", "", err
	}
	// create argo cluster
	argoclustername, err := h.createArgoCluster(ctx, envdetails.ClusterName, envdetails.ClusterKubeConfig)
	if err != nil {
		return "", "", "", err
	}
	// create argo project for env
	argoproject, err := h.createArgoProjectForEnvironment(ctx, ref, argoclustername, envdetails.Namespace)
	if err != nil {
		return "", "", "", err
	}
	return argoclustername, argoproject.Name, envdetails.Namespace, nil
}

func (h *ApplicationProcessor) deployArgoApplication(
	ctx context.Context, ref PathRef, update func(*v1alpha1.Application) error,
) (*v1alpha1.Application, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "argo-deploy")
	defer span.Finish()

	clustername, projectname, namespace, err := h.prepareArgoApplication(ctx, ref)
	if err != nil {
		return nil, err
	}

	// create argo application
	argoapplication := &v1alpha1.Application{
		TypeMeta: v1.TypeMeta{
			Kind:       application.ApplicationKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: ref.FullName(),
			Labels: map[string]string{
				LabelApplication: ref.Name,
				LabelTenant:      ref.Tenant,
				LabelProject:     ref.Project,
				LabelEnvironment: ref.Env,
			},
			Annotations: map[string]string{
				AnnotationKeyCreator: AuthorFromContext(ctx).Name,
			},
			Finalizers: []string{
				v1alpha1.ResourcesFinalizerName, // 设置级联删除策略
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Destination: v1alpha1.ApplicationDestination{
				Name:      clustername, // managed cluster name（agent name）
				Namespace: namespace,
			},
			Project: projectname,
		},
	}

	// custome update
	if err := update(argoapplication); err != nil {
		return nil, err
	}

	// 这里可能涉及到argo app的更新,使用创建或者更新
	existargoapp, err := h.Argo.EnsureArgoApp(ctx, argoapplication)
	if err != nil {
		return nil, err
	}
	return existargoapp, nil
}

func (h *ApplicationProcessor) Remove(ctx context.Context, ref PathRef) error {
	manifest, err := h.Manifest.Get(ctx, ref)
	if err != nil {
		return err
	}

	// 删除 argo app
	if err := h.Argo.RemoveArgoApp(ctx, ref.FullName()); err != nil {
		if !errors.IsNotFound(err) && grpcstatus.Code(err) != codes.NotFound {
			return err
		}
	}
	// 删除 git content
	if err := h.Manifest.Remove(ctx, ref); err != nil {
		return err
	}

	// sync database
	if err := h.DataBase.RemoveDeploy(ctx, ref, DeploiedManifest{Manifest: *manifest}); err != nil {
		log.Error(err, "remove database failed")
	}

	return nil
}

type UpdateImageArgs struct {
	Name         string   `json:"name"`
	Images       []string `json:"images,omitempty"`
	IstioVersion string   `json:"istioVersion,omitempty"`
}

func (p *ApplicationProcessor) BatchUpdateImages(ctx context.Context, ref PathRef, args []UpdateImageArgs) error {
	// 先使用一个任务更新 git
	// 如果成功，则为这些应用分别创建一个 sync 的任务
	// 如果失败，则为这些应用创建一个失败的任务
	ref.Name = ""
	err := p.Manifest.Func(ctx, ref,
		Pull(),
		FsFunc(
			// 对每个需要更新的 app 更新镜像
			func(ctx context.Context, fs billy.Filesystem) error {
				for _, item := range args {
					if item.Name == "" {
						continue
					}
					_ = fs.MkdirAll(item.Name, os.ModePerm)
					basedfs := chroot.New(fs, item.Name)

					store := NewGitFsStore(basedfs)
					UpdateContentImages(ctx, store, item.Images, item.IstioVersion)
				}
				return nil
			},
		),
		Commit("batch update images"),
	)
	// 根据结果产生新的tasks
	gitStep := workflow.Step{
		Name:   "update-image(background)",
		Status: &workflow.TaskStatus{StartTimestamp: v1.Now(), FinishTimestamp: v1.Now(), Status: workflow.TaskStatusSuccess},
	}
	if err != nil {
		gitStep.Status.Status = workflow.TaskStatusError
		gitStep.Status.Message = err.Error()
	}

	envdetails, err := p.DataBase.GetEnvironmentWithCluster(ref)
	if err != nil {
		return err
	} else {
		// 注入 cluster namespace
		ctx = context.WithValue(ctx, contextClusterNamespaceKey{}, ClusterNamespace{
			Cluster:   envdetails.ClusterName,
			Namespace: envdetails.Namespace,
		})
	}

	eg := errgroup.Group{}
	for _, arg := range args {
		iref := PathRef{Tenant: ref.Tenant, Project: ref.Project, Env: ref.Env, Name: arg.Name}

		step := gitStep
		// 便于debug
		step.Args = workflow.ArgsOf(iref, arg.Images, arg.IstioVersion)

		steps := []workflow.Step{
			step,
			{
				Name:     "sync",
				Function: TaskFunction_Application_Sync,
				Args:     workflow.ArgsOf(iref),
			},
		}

		eg.Go(func() error {
			return p.Task.SubmitTask(ctx, iref, "update-image(batch)", steps)
		})
	}
	eg.Wait()
	// 本次任务结果原样返回
	return err
}

func UpdateContentImages(ctx context.Context, store GitStore, images []string, version string) error {
	objects, err := store.ListAll(ctx)
	if err != nil {
		return err
	}
	for _, obj := range objects {
		updated := false
		ObjectPodTemplateFunc(obj, func(template *corev1.PodTemplateSpec) {
			// 更新镜像
			UpdateImage(template, images)
			// 更新 version
			template.Labels[LabelIstioVersion] = version
			updated = true
		})
		if updated {
			if err := store.Update(ctx, obj); err != nil {
				return err
			}
		}
	}
	return nil
}

func UpdateImage(template *corev1.PodTemplateSpec, images []string) {
	// 更新镜像
	for i, c := range template.Spec.Containers {
		for _, image := range images {
			// Cautious:
			//  foo/bar Match foo/bar:v1
			//  foo/bar:v1 !Match foo/bar
			if v1alpha1.KustomizeImage(c.Image).Match(v1alpha1.KustomizeImage(image)) {
				template.Spec.Containers[i].Image = image
			}
		}
	}
}
