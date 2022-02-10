package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
	"kubegems.io/pkg/agent/cluster"
)

const ChangeCauseAnnotation = "kubernetes.io/change-cause"

type RevisionHistory struct {
	Revision   int64       `json:"revision"`
	CreateTime metav1.Time `json:"createTime"`
	Namespace  string      `json:"namespace"`
	Name       string      `json:"name"`
	Kind       string      `json:"kind"`
	Images     []string    `json:"images"`
	Cause      string      `json:"cause"`
	Extra      string      `json:"extra"`
	Current    bool        `json:"current"`
}

type Revisions []*RevisionHistory

type RolloutHandler struct {
	cluster cluster.Interface
}

func (h *RolloutHandler) DaemonSetHistory(c *gin.Context) {
	h.history(c, "DaemonSet")
}

func (h *RolloutHandler) StatefulSetHistory(c *gin.Context) {
	h.history(c, "StatefulSet")
}

func (h *RolloutHandler) DeploymentHistory(c *gin.Context) {
	h.history(c, "Deployment")
}

func (h *RolloutHandler) DeploymentRollback(c *gin.Context) {
	h.rollback(c, "Deployment")
}

func (h *RolloutHandler) DaemonsetRollback(c *gin.Context) {
	h.rollback(c, "DaemonSet")
}

func (h *RolloutHandler) StatefulSetRollback(c *gin.Context) {
	h.rollback(c, "StatefulSet")
}

var annotationsToSkip = map[string]bool{
	corev1.LastAppliedConfigAnnotation:       true,
	deploymentutil.RevisionAnnotation:        true,
	deploymentutil.RevisionHistoryAnnotation: true,
	deploymentutil.DesiredReplicasAnnotation: true,
	deploymentutil.MaxReplicasAnnotation:     true,
	appsv1.DeprecatedRollbackTo:              true,
}

func (h *RolloutHandler) history(c *gin.Context, kind string) {
	ctx := c.Request.Context()
	namespace := c.Param("namespace")
	name := c.Param("name")
	switch kind {
	case "DaemonSet":
		ds, accessor, historyList, err := daemonsetHistory(ctx, h.cluster.Kubernetes(), namespace, name)
		if err != nil {
			NotOK(c, err)
			return
		}
		ret := h.reversions(historyList, accessor, namespace, name, kind, ds)
		OK(c, ret)
	case "StatefulSet":
		sts, accessor, historyList, err := statefulsetHistory(ctx, h.cluster.Kubernetes(), namespace, name)
		if err != nil {
			NotOK(c, err)
			return
		}
		ret := h.reversions(historyList, accessor, namespace, name, kind, sts)
		OK(c, ret)
	case "Deployment":
		deployment, allRSs, err := deploymentHistory(ctx, h.cluster.Kubernetes(), namespace, name)
		if err != nil {
			NotOK(c, err)
			return
		}
		current, _ := deploymentutil.Revision(deployment)
		ret := Revisions{}
		for _, rs := range allRSs {
			v, err := deploymentutil.Revision(rs)
			if err != nil {
				continue
			}
			ret = append(ret, &RevisionHistory{
				Namespace:  namespace,
				Name:       name,
				Kind:       kind,
				Images:     images(&rs.Spec.Template),
				Revision:   v,
				Current:    current == v,
				CreateTime: rs.CreationTimestamp,
				Extra:      rs.Name,
				Cause:      getChangeCuase(rs),
			})
		}
		sort.Sort(ret)
		OK(c, ret)
	}
}

func (h *RolloutHandler) reversions(historyList *appsv1.ControllerRevisionList, accessor metav1.Object, namespace, name, kind string, obj runtime.Object) Revisions {
	var (
		result     Revisions
		currentRev int64
	)
	switch o := obj.(type) {
	case *appsv1.DaemonSet:
		currentRev = o.Generation
	case *appsv1.StatefulSet:
		currentRev = o.Generation
	}
	for idx := range historyList.Items {
		rev := historyList.Items[idx]
		if metav1.IsControlledBy(&rev, accessor) {
			result = append(result, &RevisionHistory{
				Namespace:  namespace,
				Kind:       kind,
				Name:       name,
				Images:     imagesRaw(rev.Data, obj),
				Extra:      rev.Name,
				CreateTime: rev.CreationTimestamp,
				Revision:   rev.Revision,
				Current:    currentRev == rev.Revision,
			})
		}
	}
	sort.Sort(result)
	return result
}

func (h *RolloutHandler) rollback(c *gin.Context, kind string) {
	ctx := c.Request.Context()
	namespace := c.Param("namespace")
	name := c.Param("name")
	revision := c.Query("revision")
	if len(revision) == 0 {
		NotOK(c, fmt.Errorf("revision must specify"))
		return
	}
	targetRevision, err := strconv.Atoi(revision)
	if err != nil {
		NotOK(c, fmt.Errorf("revision must specify"))
		return
	}
	patchOptions := metav1.PatchOptions{}
	switch kind {
	case "Deployment":
		deploy, rslist, err := deploymentHistory(ctx, h.cluster.Kubernetes(), namespace, name)
		if err != nil {
			NotOK(c, err)
			return
		}
		revisionData := getTargetDeployRevision(rslist, int64(targetRevision))
		if revisionData == nil {
			NotOK(c, fmt.Errorf("can't find valid target revision"))
			return
		}
		if int64(targetRevision) == deploy.Generation {
			OK(c, nil)
			return
		}
		delete(revisionData.Spec.Template.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
		annotations := map[string]string{}
		for k := range annotationsToSkip {
			if v, ok := deploy.Annotations[k]; ok {
				annotations[k] = v
			}
		}
		for k, v := range revisionData.Annotations {
			if !annotationsToSkip[k] {
				annotations[k] = v
			}
		}
		patchType, patch, err := getDeploymentPatch(&revisionData.Spec.Template, annotations)
		if err != nil {
			NotOK(c, fmt.Errorf("failed to restore rs %v", err))
			return
		}
		if _, err = h.cluster.Kubernetes().AppsV1().Deployments(namespace).Patch(context.TODO(), name, patchType, patch, patchOptions); err != nil {
			NotOK(c, fmt.Errorf("failed to rollback rs %v", err))
			return
		}
		OK(c, nil)
	case "DaemonSet":
		ds, _, historyList, err := daemonsetHistory(ctx, h.cluster.Kubernetes(), namespace, name)
		if err != nil {
			NotOK(c, err)
			return
		}
		revisionData := getToRevision(historyList, int64(targetRevision))
		if revisionData == nil {
			NotOK(c, fmt.Errorf("can't find valid target revision"))
			return
		}
		if revisionData.Revision == ds.Generation {
			OK(c, nil)
			return
		}
		if _, err = h.cluster.Kubernetes().AppsV1().DaemonSets(ds.Namespace).Patch(ctx, ds.Name, types.StrategicMergePatchType, revisionData.Data.Raw, patchOptions); err != nil {
			NotOK(c, err)
		}
		OK(c, nil)
	case "StatefulSet":
		sts, _, historyList, err := statefulsetHistory(ctx, h.cluster.Kubernetes(), namespace, name)
		if err != nil {
			NotOK(c, err)
			return
		}
		revisionData := getToRevision(historyList, int64(targetRevision))
		if revisionData == nil {
			NotOK(c, fmt.Errorf("can't find valid target revision"))
			return
		}
		if revisionData.Revision == sts.Generation {
			OK(c, nil)
			return
		}
		if _, err = h.cluster.Kubernetes().AppsV1().StatefulSets(sts.Namespace).Patch(ctx, sts.Name, types.StrategicMergePatchType, revisionData.Data.Raw, patchOptions); err != nil {
			NotOK(c, err)
		}
		OK(c, nil)
	}
}

func getChangeCuase(accessor metav1.Object) string {
	v, exist := accessor.GetAnnotations()[ChangeCauseAnnotation]
	if exist {
		return v
	}
	return ""
}

func (r Revisions) Len() int {
	return len(r)
}

func (r Revisions) Less(i, j int) bool {
	return r[i].Revision > r[j].Revision
}

func (r Revisions) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func daemonsetHistory(ctx context.Context, k kubernetes.Interface, namespace, name string) (ds *appsv1.DaemonSet, accessor metav1.Object, historyList *appsv1.ControllerRevisionList, err error) {
	ds, err = k.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return
	}
	selector, err := metav1.LabelSelectorAsSelector(ds.Spec.Selector)
	if err != nil {
		return
	}
	if err != nil {
		return
	}
	accessor, err = meta.Accessor(ds)
	if err != nil {
		return
	}
	historyList, err = k.AppsV1().ControllerRevisions(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	return
}

func statefulsetHistory(ctx context.Context, k kubernetes.Interface, namespace, name string) (sts *appsv1.StatefulSet, accessor metav1.Object, historyList *appsv1.ControllerRevisionList, err error) {
	sts, err = k.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return
	}
	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		return
	}
	accessor, err = meta.Accessor(sts)
	if err != nil {
		return
	}
	historyList, err = k.AppsV1().ControllerRevisions(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	return
}

func deploymentHistory(ctx context.Context, k kubernetes.Interface, namespace, name string) (deployment *appsv1.Deployment, rsList []*appsv1.ReplicaSet, err error) {
	deployment, err = k.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return
	}
	_, allOldRSs, newRS, err := deploymentutil.GetAllReplicaSets(deployment, k.AppsV1())
	if err != nil {
		return
	}
	rsList = allOldRSs
	if newRS != nil {
		rsList = append(allOldRSs, newRS)
	}
	return
}

func getToRevision(list *appsv1.ControllerRevisionList, targetRevision int64) *appsv1.ControllerRevision {
	for idx := range list.Items {
		tmp := list.Items[idx]
		if tmp.Revision == targetRevision {
			return &tmp
		}
	}
	return nil
}

func getTargetDeployRevision(list []*appsv1.ReplicaSet, targetRevision int64) *appsv1.ReplicaSet {
	for idx := range list {
		tmp := list[idx]
		v, err := deploymentutil.Revision(tmp)
		if err != nil {
			continue
		}
		if v == targetRevision {
			return tmp
		}
	}
	return nil
}

func getDeploymentPatch(podTemplate *corev1.PodTemplateSpec, annotations map[string]string) (types.PatchType, []byte, error) {
	patch, err := json.Marshal([]interface{}{
		map[string]interface{}{
			"op":    "replace",
			"path":  "/spec/template",
			"value": podTemplate,
		},
		map[string]interface{}{
			"op":    "replace",
			"path":  "/metadata/annotations",
			"value": annotations,
		},
	})
	return types.JSONPatchType, patch, err
}

func images(podTemplate *corev1.PodTemplateSpec) []string {
	ret := []string{}
	for _, c := range podTemplate.Spec.Containers {
		ret = append(ret, c.Image)
	}
	return ret
}

func imagesRaw(raw runtime.RawExtension, obj runtime.Object) []string {
	origin, err := json.Marshal(obj)
	if err != nil {
		return []string{}
	}
	ret, err := strategicpatch.StrategicMergePatch(origin, raw.Raw, obj)
	if err != nil {
		return []string{}
	}
	otmp := obj.DeepCopyObject()
	json.Unmarshal(ret, otmp)
	switch o := otmp.(type) {
	case *appsv1.DaemonSet:
		return images(&o.Spec.Template)
	case *appsv1.StatefulSet:
		return images(&o.Spec.Template)
	default:
		return []string{}
	}
}
