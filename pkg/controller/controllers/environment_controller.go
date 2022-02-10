/*
Copyright 2021 cloudminds.com.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/controller/utils"
	gemlabels "kubegems.io/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	imagePullSecretKeyPrefix = "kubegems.io/imagePullSecrets-"
)

// EnvironmentReconciler reconciles a Environment object
type EnvironmentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=kubegems.io,resources=environments,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=kubegems.io,resources=environments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=limitranges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete

func (r *EnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		环境逻辑：
		1. 创建,更新或者关联Namespace, 给namespace打标签
		2. 创建或者更新ResourceQuota,打标签
		3. 创建或者更新LimitRange,打标签
		4. 创建的时候，添加finalizer;删除得时候，根据策略删除对应的ns,或者删除label
	*/
	log := r.Log.WithName("Environment").WithValues("Environment", req.Name)
	var env gemsv1beta1.Environment
	if err := r.Get(ctx, req.NamespacedName, &env); err != nil {
		log.Info("Faild to get Environment")
		return ctrl.Result{}, nil
	}

	nsLabel := map[string]string{
		gemlabels.LabelProject:     env.Spec.Project,
		gemlabels.LabelTenant:      env.Spec.Tenant,
		gemlabels.LabelEnvironment: env.Name,
	}

	owner := metav1.NewControllerRef(&env, gemsv1beta1.SchemeEnvironment)

	// 删除环境
	if !env.ObjectMeta.DeletionTimestamp.IsZero() {
		// 根据删除策略，删除对应ns或者去掉ns下的label
		if controllerutil.ContainsFinalizer(&env, gemlabels.FinalizerNamespace) {
			if err := r.handleDelete(&env, nsLabel, owner, ctx, log); err != nil {
				log.Error(err, "Faild to delete Environment related namespace")
				return ctrl.Result{Requeue: true}, err
			}
			controllerutil.RemoveFinalizer(&env, gemlabels.FinalizerNamespace)
			if err := r.Update(ctx, &env); err != nil {
				log.Error(err, "Faild to delete Environment finalizer")
				return ctrl.Result{Requeue: true}, err
			}
		}
		return ctrl.Result{}, nil
	}

	//	处理关联namespace
	r.handleNamespace(&env, nsLabel, ctx, log)

	//	处理关联resourceQuota
	r.handleResourceQuota(&env, nsLabel, ctx, log)

	//	处理关联limitrage
	r.handleLimitRange(&env, nsLabel, ctx, log)

	// 更新环境中的serviceaccount
	r.handleServiceAccount(&env, nsLabel, ctx, log)

	var changed bool
	if utils.LabelChanged(env.Labels, nsLabel) {
		env.Labels = labels.Merge(env.Labels, nsLabel)
		changed = true
	}
	if !controllerutil.ContainsFinalizer(&env, gemlabels.FinalizerNamespace) {
		controllerutil.AddFinalizer(&env, gemlabels.FinalizerNamespace)
		changed = true
	}
	if changed {
		r.Update(ctx, &env)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gemsv1beta1.Environment{}).
		Complete(r)
}

func (r *EnvironmentReconciler) handleNamespace(env *gemsv1beta1.Environment, nlabels map[string]string, ctx context.Context, log logr.Logger) {
	var ns corev1.Namespace
	nsKey := types.NamespacedName{
		Name: env.Spec.Namespace,
	}
	if err := r.Get(ctx, nsKey, &ns); err != nil {
		if errors.IsNotFound(err) {
			ns.Name = env.Spec.Namespace
			if err := controllerutil.SetControllerReference(env, &ns, r.Scheme); err != nil {
				log.Info("failed to set controllerRef")
				return
			}
			ns.Labels = labels.Merge(ns.Labels, nlabels)
			if err := r.Create(ctx, &ns); err != nil {
				r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedCreate, "Failed to create namespace %s: %v", env.Spec.Namespace, err)
				log.Info("Error create namespace")
				return
			}
			r.Recorder.Eventf(env, corev1.EventTypeNormal, utils.ReasonCreated, "Successfully create namespace %s", env.Spec.Namespace)
		} else {
			r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonUnknowError, "Failed relate namespace %s to Environment %s: %v", env.Spec.Namespace, env.Name, err)
			log.Error(err, "Error get namespace")
			return
		}
	}

	var changed bool
	// NOTICE: [bugfix] 当修改环境关联的namespace的时候，那么之前的namespace的所有label和owerReferences需要删除，不然在删除的时候会被级联删除掉
	originNsList := corev1.NamespaceList{}
	r.List(ctx, &originNsList, client.MatchingLabels(nlabels))
	for idx := range originNsList.Items {
		if originNsList.Items[idx].Name == ns.Name {
			continue
		}
		utils.DeleteLabels(originNsList.Items[idx].Labels, nlabels)
		originNsList.Items[idx].ObjectMeta.OwnerReferences = nil
		r.Update(ctx, &originNsList.Items[idx])
	}

	if metav1.GetControllerOf(&ns) == nil {
		controllerutil.SetControllerReference(env, &ns, r.Scheme)
		changed = true
	}
	if utils.LabelChanged(ns.Labels, nlabels) {
		ns.Labels = labels.Merge(ns.Labels, nlabels)
		changed = true
	}
	if changed {
		if err := r.Update(ctx, &ns); err != nil {
			r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedUpdate, "Failed to update Namespace %s belong to Environment %s", env.Spec.Namespace, env.Name)
			log.Error(err, "Error update namespace")
		}
		r.Recorder.Eventf(env, corev1.EventTypeNormal, utils.ReasonUpdated, "Successfully update namespace %s belong to Environment %s", env.Spec.Namespace, env.Name)
	}
}

func (r *EnvironmentReconciler) handleResourceQuota(env *gemsv1beta1.Environment, nlabels map[string]string, ctx context.Context, log logr.Logger) {
	var nsrq corev1.ResourceQuota
	nsrqkey := types.NamespacedName{
		Namespace: env.Spec.Namespace,
		Name:      env.Spec.ResourceQuotaName,
	}
	if err := r.Get(ctx, nsrqkey, &nsrq); err != nil {
		if errors.IsNotFound(err) {
			nsrq.Name = env.Spec.ResourceQuotaName
			nsrq.Namespace = env.Spec.Namespace
			nsrq.Spec.Hard = env.Spec.ResourceQuota
			controllerutil.SetControllerReference(env, &nsrq, r.Scheme)
			nsrq.Labels = labels.Merge(nsrq.Labels, nlabels)
			if err := r.Create(ctx, &nsrq); err != nil {
				r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedCreate, "Failed to create ResourceQuota %s: %v", env.Spec.ResourceQuotaName, err)
				log.Info("Error create resourceQuota")
				return
			}
			r.Recorder.Eventf(env, corev1.EventTypeNormal, utils.ReasonCreated, "Successfully create ResourceQuota %s", env.Spec.ResourceQuotaName)
		} else {
			r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonUnknowError, "Failed relate ResourceQuota %s to Environment %s: %v", env.Spec.ResourceQuotaName, env.Name, err)
			log.Error(err, "Error get resourceQuota")
			return
		}
	}

	var changed bool
	if !equality.Semantic.DeepEqual(nsrq.Spec.Hard, env.Spec.ResourceQuota) {
		nsrq.Spec.Hard = env.Spec.ResourceQuota
		nsrq.Labels = labels.Merge(nsrq.Labels, nlabels)
		changed = true
	}
	if metav1.GetControllerOf(&nsrq) == nil {
		controllerutil.SetControllerReference(env, &nsrq, r.Scheme)
		changed = true
	}
	if changed {
		if err := r.Update(ctx, &nsrq); err != nil {
			r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedUpdate, "Failed to update ResourceQuota %s belong to Environment %s: %v", env.Spec.ResourceQuotaName, env.Name, err)
			log.Info("Error update resourceQuota")
		}
		r.Recorder.Eventf(env, corev1.EventTypeNormal, utils.ReasonCreated, "Successfully updated ResourceQuota %s to Environment %s", env.Spec.ResourceQuotaName, env.Name)
	}
}

func (r *EnvironmentReconciler) handleLimitRange(env *gemsv1beta1.Environment, nlabels map[string]string, ctx context.Context, log logr.Logger) {
	var lr corev1.LimitRange
	lrkey := types.NamespacedName{
		Namespace: env.Spec.Namespace,
		Name:      env.Spec.LimitRageName,
	}
	if err := r.Get(ctx, lrkey, &lr); err != nil {
		if errors.IsNotFound(err) {
			lr.Name = env.Spec.LimitRageName
			lr.Namespace = env.Spec.Namespace
			lr.Labels = labels.Merge(lr.Labels, nlabels)
			lr.Spec.Limits = env.Spec.LimitRage
			if err := r.Create(ctx, &lr); err != nil {
				r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedCreate, "Failed to create LimitRange %s: %v", env.Spec.LimitRageName, err)
				log.Info("Error create limitrange " + err.Error())
				return
			}
			r.Recorder.Eventf(env, corev1.EventTypeNormal, utils.ReasonCreated, "Successfully create LimitRange %s", env.Spec.LimitRageName)
		} else {
			r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonUnknowError, "Failed relate LimitRange %s to Environment %s: %v", env.Spec.LimitRageName, env.Name, err)
			log.Error(err, "Error get limitrange")
			return
		}
	}
	var changed bool
	if !equality.Semantic.DeepEqual(lr.Spec.Limits, env.Spec.LimitRage) {
		lr.Spec.Limits = env.Spec.LimitRage
		lr.Labels = labels.Merge(lr.Labels, nlabels)
		changed = true
	}
	if metav1.GetControllerOf(&lr) == nil {
		controllerutil.SetControllerReference(env, &lr, r.Scheme)
		changed = true
	}
	if changed {
		if err := r.Update(ctx, &lr); err != nil {
			r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedUpdate, "Failed to update LimitRange %s belong to Environment %s", env.Spec.LimitRageName, env.Name)
			log.Info("Error update limigrange")
		}
		r.Recorder.Eventf(env, corev1.EventTypeNormal, utils.ReasonUpdated, "Successfully update LimitRange %s belong to Environment %s", env.Spec.LimitRageName, env.Name)
	}
}

func (r *EnvironmentReconciler) handleServiceAccount(env *gemsv1beta1.Environment, nlabels map[string]string, ctx context.Context, log logr.Logger) {
	serviceAccountList := corev1.ServiceAccountList{}
	if err := r.Client.List(ctx, &serviceAccountList, &client.ListOptions{Namespace: env.Spec.Namespace}); err != nil {
		msg := fmt.Sprintf("failed to list serviceaccount in namespace %s", env.Spec.Namespace)
		r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedUpdate, msg)
		r.Log.Error(err, msg)
	}
	saMap := map[string]corev1.ServiceAccount{} // map加快匹配
	for _, v := range serviceAccountList.Items {
		saMap[v.Name] = v
	}

	for k := range env.Annotations {
		if strings.HasPrefix(k, imagePullSecretKeyPrefix) {
			saName := strings.TrimPrefix(k, imagePullSecretKeyPrefix)
			if sa, ok := saMap[saName]; ok {
				secrets, hasDiff := imagePullSecretHasDiff(sa.ImagePullSecrets, env.Annotations[k])
				if hasDiff {
					sa.ImagePullSecrets = secrets
					if err := r.Client.Update(ctx, &sa); err != nil {
						msg := fmt.Sprintf("failed to update serviceaccount %v", sa)
						r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedUpdate, msg)
						r.Log.Error(err, msg)
					}
					r.Log.Info(fmt.Sprintf("success to update serviceaccount %s in namespace %s", sa.Name, sa.Namespace))
				}
			}
		}
	}
}

func imagePullSecretHasDiff(obj []corev1.LocalObjectReference, str string) ([]corev1.LocalObjectReference, bool) {
	oldSec := make([]string, len(obj))
	for i := range obj {
		oldSec[i] = obj[i].Name
	}
	newSec := strings.Split(str, ",")
	for i := range newSec {
		newSec[i] = strings.TrimSpace(newSec[i]) // 去除空格
	}

	if !utils.StringArrayEqual(oldSec, newSec) {
		ret := make([]corev1.LocalObjectReference, len(newSec))
		for i := range newSec {
			ret[i].Name = newSec[i]
		}
		return ret, true
	}

	return nil, false
}

func (r *EnvironmentReconciler) handleDelete(env *gemsv1beta1.Environment, todelLabels map[string]string, owner *metav1.OwnerReference, ctx context.Context, log logr.Logger) error {
	var ns corev1.Namespace
	nsKey := types.NamespacedName{
		Name: env.Spec.Namespace,
	}
	if err := r.Get(ctx, nsKey, &ns); err != nil {
		if i := client.IgnoreNotFound(err); i != nil {
			return i
		}
	}
	if env.Spec.DeletePolicy == gemsv1beta1.DeletePolicyDelLabels {
		ns.ObjectMeta.Labels = utils.DeleteLabels(ns.ObjectMeta.Labels, todelLabels)
		ns.SetOwnerReferences(nil)
		if err := r.Update(ctx, &ns); err != nil {
			r.Recorder.Eventf(env, corev1.EventTypeWarning, utils.ReasonFailedDelete, "Failed to delete environment labels for namespace %s", env.Spec.Namespace)
			return err
		}
		r.Recorder.Eventf(env, corev1.EventTypeNormal, utils.ReasonDeleted, "Successfully to delete environment labels for namespace %s", env.Spec.Namespace)
	}
	return nil
}
