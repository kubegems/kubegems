package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5/plumbing/object"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/workflow"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

const LabelIstioVersion = "version"

var ForFileContentFunc = git.ForFileContentFunc

var KustimizationFilename = konfig.DefaultKustomizationFileName()

func InitOrUpdateKustomization(fs billy.Filesystem, labels, annotations map[string]string) error {
	fis, err := fs.ReadDir(".")
	if err != nil {
		return err
	}

	// 如果文件系统上没有文件，则是被删除的应用，此时不生成kustomize
	if len(fis) == 0 {
		return nil
	}

	resourcefiles := []string{}
	kustomization := &types.Kustomization{}

	_ = ForFileContentFunc(fs, "", func(filename string, content []byte) error {
		if filepath.Ext(filename) != ".yaml" {
			return nil
		}
		if filename == KustimizationFilename {
			// 如果yaml文件格式不正确，在下一步写入正确的文件内容
			if err := yaml.Unmarshal(content, &kustomization); err != nil {
				log.Errorf("invalid kustomization file %s,override it next", filename)
			}
			return nil
		}
		// 其余的均视为资源文件
		// fullpath 是文件系统上的路径，作为 resources 时需要去除base路径
		resourcefiles = append(resourcefiles, filename)
		return nil
	})

	// 写入/更新 kustomization.yaml
	kustomization.FixKustomizationPostUnmarshalling()
	kustomization.Resources = resourcefiles

	if kustomization.CommonLabels == nil {
		kustomization.CommonLabels = map[string]string{}
	}
	for k, v := range labels {
		kustomization.CommonLabels[k] = v
	}

	// ignore labels k "version"
	// 不允许设置 istio version 作为 commonLabels，会导致未知问题
	delete(kustomization.CommonLabels, LabelIstioVersion)

	if kustomization.CommonAnnotations == nil {
		kustomization.CommonAnnotations = map[string]string{}
	}
	for k, v := range annotations {
		kustomization.CommonAnnotations[k] = v
	}

	kustomizecontent, err := yaml.Marshal(kustomization)
	if err != nil {
		return err
	}
	return util.WriteFile(fs, KustimizationFilename, kustomizecontent, os.ModePerm)
}

var decoder = serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()

func DecodeResource(content []byte) (client.Object, error) {
	obj, gvk, err := decoder.Decode(content, nil, nil)
	if err != nil {
		return nil, err
	}

	if cobj, ok := obj.(client.Object); ok {
		obj.GetObjectKind().SetGroupVersionKind(*gvk)
		return cobj, nil
	}
	return nil, nil
}

func ObjectPodTemplateFunc(obj client.Object, fun func(template *corev1.PodTemplateSpec)) {
	var template *corev1.PodTemplateSpec
	switch data := obj.(type) {
	case *appsv1.Deployment:
		template = &data.Spec.Template
	case *appsv1.DaemonSet:
		template = &data.Spec.Template
	case *appsv1.StatefulSet:
		template = &data.Spec.Template
	case *appsv1.ReplicaSet:
		template = &data.Spec.Template
	case *batchv1.Job:
		template = &data.Spec.Template
	case *batchv1beta1.CronJob:
		template = &data.Spec.JobTemplate.Spec.Template
	default:
		return
	}
	fun(template)
}

type (
	contextAuthorKey           struct{}
	contextClusterNamespaceKey struct{}
)

func AuthorFromContext(ctx context.Context) *object.Signature {
	if author, ok := ctx.Value(contextAuthorKey{}).(*object.Signature); ok {
		return author
	}
	// 如果是在异步任务中运行
	if val := workflow.ValueFromConetxt(ctx, TaskAddtionalKeyCommiter); val != "" {
		return &object.Signature{Name: val, Email: val, When: time.Now()}
	}
	return &object.Signature{Name: "unknown", Email: "unknown", When: time.Now()}
}

func ClusterNamespaceFromCtx(ctx context.Context) (string, string) {
	if val, ok := ctx.Value(contextClusterNamespaceKey{}).(ClusterNamespace); ok {
		return val.Cluster, val.Namespace
	}
	return "", ""
}

func ParseMainDeployment(ctx context.Context, store GitStore) (*appsv1.Deployment, error) {
	// deployment
	deploymentList := &appsv1.DeploymentList{}
	if err := store.List(ctx, deploymentList); err != nil {
		return nil, err
	}
	if len(deploymentList.Items) != 1 {
		return nil, fmt.Errorf("none or more than one deployment found")
	}
	deployment := &deploymentList.Items[0]
	return deployment, nil
}

func ParseMainStatefulSet(ctx context.Context, store GitStore) (*appsv1.StatefulSet, error) {
	// statefulset
	statefulSetList := &appsv1.StatefulSetList{}
	if err := store.List(ctx, statefulSetList); err != nil {
		return nil, err
	}
	if len(statefulSetList.Items) != 1 {
		return nil, fmt.Errorf("none or more than one statefulset found")
	}
	return &statefulSetList.Items[0], nil
}

func ParseMainWorkload(ctx context.Context, store GitStore) (client.Object, error) {
	objects, err := store.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return mainWorkloadByPriority(objects), nil
}

func mainWorkloadByPriority(objects []client.Object) client.Object {
	var ret client.Object
	priority := -1
	for _, obj := range objects {
		if p, ok := kindPriorityMap[obj.GetObjectKind().GroupVersionKind().Kind]; ok && p > priority {
			ret = obj
			priority = p
		}
	}
	return ret
}
