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

package noproxy

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/agents"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const apiVersion = "apps/v1"

// cpu memory 表示需要配置的百分比
// real_cpu real_memory 表示真实的hpa上的值，反向计算出来的百分比
type hpaForm struct {
	Cluster     string `json:"cluster" binding:"required"`
	Kind        string `json:"kind" binding:"required,eq=StatefulSet|eq=Deployment"`
	Namespace   string `json:"namespace" binding:"required"`
	Name        string `json:"name" binding:"required"`
	MinReplicas int32  `json:"min_replicas" binding:"required,gte=1"`
	MaxReplicas int32  `json:"max_replicas" binding:"required"`
	Cpu         int32  `json:"cpu" binding:"lte=100"`
	Memory      int32  `json:"memory" binding:"lte=100"`
	RealCpu     int32  `json:"real_cpu"`
	RealMemory  int32  `json:"real_memory"`
	Exist       bool   `json:"exist"`
}

type hpaQuery struct {
	Kind string `json:"kind" form:"kind"`
	Name string `json:"name" form:"name"`
}

// @Tags			NOPROXY
// @Summary		设置HPA
// @Description	设置HPA
// @Accept			json
// @Produce		json
// @Param			cluster		path		string									true	"cluster"
// @Param			namespace	path		string									true	"namespace"
// @Param			param		body		hpaForm									true	"表单"
// @Success		200			{object}	handlers.ResponseStruct{Data=object}	"object"
// @Router			/v1/noproxy/{cluster}/{namespace}/hpa [post]
// @Security		JWT
func (h *HpaHandler) SetObjectHpa(c *gin.Context) {
	form := &hpaForm{}
	if err := c.BindJSON(form); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if form.Cpu == 0 && form.Memory == 0 {
		handlers.NotOK(c, i18n.Errorf(c, "CPU or Memory can't be empty at the same time, please provide one at least"))
		return
	}

	action := i18n.Sprintf(context.TODO(), "set")
	module := i18n.Sprintf(context.TODO(), "HPA")
	h.SetAuditData(c, action, module, fmt.Sprintf("%s/%s", form.Namespace, form.Name))
	h.SetExtraAuditDataByClusterNamespace(c, form.Cluster, form.Namespace)
	hpa, err := h.createOrUpdateHPA(c.Request.Context(), form)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, hpa)
}

// @Tags			NOPROXY
// @Summary		获取HPA form
// @Description	获取HPA form
// @Accept			json
// @Produce		json
// @Param			cluster		path		string									true	"cluster"
// @Param			namespace	path		string									true	"namespace"
// @Param			name		query		string									true	"name"
// @Param			cluster		query		string									true	"cluster"
// @Param			kind		query		string									true	"kind"
// @Success		200			{object}	handlers.ResponseStruct{Data=hpaForm}	"object"
// @Router			/v1/noproxy/{cluster}/{namespace}/hpa [get]
// @Security		JWT
func (h *HpaHandler) GetObjectHpa(c *gin.Context) {
	namespace := c.Param("namespace")
	cluster := c.Param("cluster")

	query := &hpaQuery{}
	if err := c.BindQuery(query); err != nil {
		return
	}
	hpaform := hpaForm{
		Kind:      query.Kind,
		Name:      query.Name,
		Cluster:   cluster,
		Namespace: namespace,
		Exist:     false,
	}
	hpaname := FormatHPAName(query.Kind, query.Name)

	hpa := &v2beta2.HorizontalPodAutoscaler{}
	err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: hpaname}, hpa)
	})
	if err != nil {
		handlers.OK(c, hpaform)
		return
	}

	lmt, req, err := h.getRealResource(c.Request.Context(), cluster, namespace, query.Name, query.Kind)
	if err != nil {
		handlers.OK(c, hpaform)
		return
	}

	hpaform.MaxReplicas = hpa.Spec.MaxReplicas
	hpaform.MinReplicas = *hpa.Spec.MinReplicas
	currentCPU, currentMemory, beforeCPU, beforeMemory := getHPAPercent(hpa, lmt, req)
	hpaform.Cpu = int32(beforeCPU)
	hpaform.Memory = int32(beforeMemory)
	hpaform.RealCpu = int32(currentCPU)
	hpaform.RealMemory = int32(currentMemory)
	hpaform.Exist = true
	handlers.OK(c, hpaform)
}

func (h *HpaHandler) getRealResource(ctx context.Context, cluster, namespace, name, kind string) (lmt v1.ResourceList, req v1.ResourceList, err error) {
	var obj client.Object
	switch kind {
	case "StatefulSet":
		obj = &appsv1.StatefulSet{}
	case "Deployment":
		obj = &appsv1.Deployment{}
	default:
		err = i18n.Errorf(ctx, "unsuppored kind %s to set/get HPA, StatefulSet and Deployment supported", kind)
		return
	}
	err = h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, obj)
	})
	if err != nil {
		return
	}
	switch kind {
	case "StatefulSet":
		lmt, req = containerResources(obj.(*appsv1.StatefulSet).Spec.Template.Spec.Containers)
	case "Deployment":
		lmt, req = containerResources(obj.(*appsv1.Deployment).Spec.Template.Spec.Containers)
	}
	return
}

func (h *HpaHandler) createOrUpdateHPA(ctx context.Context, form *hpaForm) (*v2beta2.HorizontalPodAutoscaler, error) {
	hpa := &v2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FormatHPAName(form.Kind, form.Name),
			Namespace: form.Namespace,
		},
	}

	lmt, req, err := h.getRealResource(ctx, form.Cluster, form.Namespace, form.Name, form.Kind)
	if err != nil {
		return nil, err
	}

	err = h.Execute(ctx, form.Cluster, func(ctx context.Context, cli agents.Client) error {
		_, err := controllerutil.CreateOrUpdate(ctx, cli, hpa, func() error {
			return form.Update(hpa, lmt, req)
		})
		return err
	})

	return hpa, err
}

func getHPAPercent(hpa *v2beta2.HorizontalPodAutoscaler, lmt, req v1.ResourceList) (currentCPUPercent, currentMemoryPercent, beforeCPUPercent, beforeMemoryPercent int64) {
	var (
		realCPU    int64
		realMemory int64
	)
	for _, m := range hpa.Spec.Metrics {
		if m.Resource.Name == v1.ResourceCPU {
			realCPU = int64(pointer.Int32Deref(m.Resource.Target.AverageUtilization, 0))
		}
		if m.Resource.Name == v1.ResourceMemory {
			realMemory = int64(pointer.Int32Deref(m.Resource.Target.AverageUtilization, 0))
		}
	}
	if lmt.Cpu().IsZero() {
		currentCPUPercent = 0
	} else {
		currentCPUPercent = realCPU * req.Cpu().MilliValue() / lmt.Cpu().MilliValue()
	}

	if lmt.Memory().IsZero() {
		currentMemoryPercent = 0
	} else {
		currentMemoryPercent = realMemory * req.Memory().MilliValue() / lmt.Memory().MilliValue()
	}
	beforeCPU, _ := strconv.Atoi(hpa.Annotations["cpu"])
	beforeMemory, _ := strconv.Atoi(hpa.Annotations["memory"])
	beforeCPUPercent = int64(beforeCPU)
	beforeMemoryPercent = int64(beforeMemory)
	return
}

func (form *hpaForm) Update(in *v2beta2.HorizontalPodAutoscaler, lmt, req v1.ResourceList) error {
	var metrics []v2beta2.MetricSpec
	if form.Cpu > 0 && !lmt.Cpu().IsZero() && !req.Cpu().IsZero() {
		realCPU := int32(int64(form.Cpu) * lmt.Cpu().MilliValue() / req.Cpu().MilliValue())
		metrics = append(metrics, v2beta2.MetricSpec{
			Type: v2beta2.ResourceMetricSourceType,
			Resource: &v2beta2.ResourceMetricSource{
				Name: v1.ResourceCPU,
				Target: v2beta2.MetricTarget{
					Type:               v2beta2.UtilizationMetricType,
					AverageUtilization: &realCPU,
				},
			},
		})
	}
	if form.Memory > 0 && !lmt.Memory().IsZero() && !req.Memory().IsZero() {
		realMemory := int32(int64(form.Memory) * lmt.Memory().MilliValue() / req.Memory().MilliValue())
		metrics = append(metrics, v2beta2.MetricSpec{
			Type: v2beta2.ResourceMetricSourceType,
			Resource: &v2beta2.ResourceMetricSource{
				Name: v1.ResourceMemory,
				Target: v2beta2.MetricTarget{
					Type:               v2beta2.UtilizationMetricType,
					AverageUtilization: &realMemory,
				},
			},
		})
	}

	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}
	in.Annotations["cpu"] = strconv.Itoa(int(form.Cpu))
	in.Annotations["memory"] = strconv.Itoa(int(form.Memory))

	in.Spec.Metrics = metrics
	in.Spec.MinReplicas = &form.MinReplicas
	in.Spec.MaxReplicas = form.MaxReplicas
	in.Spec.ScaleTargetRef = v2beta2.CrossVersionObjectReference{
		Kind:       form.Kind,
		Name:       form.Name,
		APIVersion: apiVersion,
	}
	return nil
}

func FormatHPAName(kind, targetName string) string {
	k := ""
	if strings.ToLower(kind) == "statefulset" {
		k = "sts"
	}
	if strings.ToLower(kind) == "deployment" {
		k = "dep"
	}
	return fmt.Sprintf("hpa-%s-%s", k, targetName)
}

func containerResources(containers []v1.Container) (v1.ResourceList, v1.ResourceList) {
	lmt := v1.ResourceList{}
	req := v1.ResourceList{}
	for _, c := range containers {
		for k, v := range c.Resources.Limits.DeepCopy() {
			if ev, exist := lmt[k]; exist {
				ev.Add(v)
				lmt[k] = ev
			} else {
				lmt[k] = v
			}
		}
		for k, v := range c.Resources.Requests.DeepCopy() {
			if ev, exist := req[k]; exist {
				ev.Add(v)
				req[k] = ev
			} else {
				req[k] = v
			}
		}
	}
	return lmt, req
}
