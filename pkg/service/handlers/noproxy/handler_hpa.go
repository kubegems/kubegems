package noproxy

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const apiVersion = "apps/v1"

type hpaForm struct {
	Cluster     string `json:"cluster" binding:"required"`
	Kind        string `json:"kind" binding:"required,eq=Statefulset|eq=Deployment"`
	Namespace   string `json:"namespace" binding:"required"`
	Name        string `json:"name" binding:"required"`
	MinReplicas int32  `json:"min_replicas" binding:"required,gte=1"`
	MaxReplicas int32  `json:"max_replicas" binding:"required"`
	Cpu         int32  `json:"cpu" binding:"lte=100"`
	Memory      int32  `json:"memory" binding:"lte=100"`
	Exist       bool   `json:"exist"`
}

type hpaQuery struct {
	Kind string `json:"kind" form:"kind"`
	Name string `json:"name" form:"name"`
}

// @Tags NOPROXY
// @Summary 设置HPA
// @Description 设置HPA
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param param body hpaForm true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "object"
// @Router /v1/noproxy/{cluster}/{namespace}/hpa [post]
// @Security JWT
func (h *HpaHandler) SetObjectHpa(c *gin.Context) {
	form := &hpaForm{}
	if err := c.BindJSON(form); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if form.Cpu == 0 && form.Memory == 0 {
		handlers.NotOK(c, fmt.Errorf("内存和CPU不可以同时为空"))
		return
	}

	h.SetAuditData(c, "配置", "HPA", fmt.Sprintf("%v/%v", form.Namespace, form.Name))
	h.SetExtraAuditDataByClusterNamespace(c, form.Cluster, form.Namespace)
	hpa, err := h.createOrUpdateHPA(c.Request.Context(), form)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, hpa)
}

// @Tags NOPROXY
// @Summary 获取HPA form
// @Description 获取HPA form
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name query string true "name"
// @Param cluster query string true "cluster"
// @Param kind query string true "kind"
// @Success 200 {object} handlers.ResponseStruct{Data=hpaForm} "object"
// @Router /v1/noproxy/{cluster}/{namespace}/hpa [get]
// @Security JWT
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

	hpa := &v2beta1.HorizontalPodAutoscaler{}
	err := h.Execute(c.Request.Context(), cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: hpaname}, hpa)
	})
	if err != nil {
		handlers.OK(c, hpaform)
		return
	}
	hpaform.MaxReplicas = hpa.Spec.MaxReplicas
	hpaform.MinReplicas = *hpa.Spec.MinReplicas
	cpu, memory := getHPAPercent(hpa)
	hpaform.Cpu = cpu
	hpaform.Memory = memory
	hpaform.Exist = true
	handlers.OK(c, hpaform)
}

func (h *HpaHandler) createOrUpdateHPA(ctx context.Context, form *hpaForm) (*v2beta1.HorizontalPodAutoscaler, error) {
	hpa := &v2beta1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FormatHPAName(form.Kind, form.Name),
			Namespace: form.Namespace,
		},
	}

	err := h.Execute(ctx, form.Cluster, func(ctx context.Context, cli agents.Client) error {
		_, err := controllerutil.CreateOrUpdate(ctx, cli, hpa, func() error {
			return form.Update(hpa)
		})
		return err
	})

	return hpa, err
}

func getHPAPercent(hpa *v2beta1.HorizontalPodAutoscaler) (int32, int32) {
	var (
		memory int32
		cpu    int32
	)
	for _, m := range hpa.Spec.Metrics {
		if m.Resource.Name == v1.ResourceCPU {
			cpu = *m.Resource.TargetAverageUtilization
		}
		if m.Resource.Name == v1.ResourceMemory {
			memory = *m.Resource.TargetAverageUtilization
		}
	}
	return cpu, memory
}

func (form *hpaForm) Update(in *v2beta1.HorizontalPodAutoscaler) error {
	var metrics []v2beta1.MetricSpec
	if form.Cpu > 0 {
		metrics = append(metrics, v2beta1.MetricSpec{
			Type: v2beta1.ResourceMetricSourceType,
			Resource: &v2beta1.ResourceMetricSource{
				Name:                     v1.ResourceCPU,
				TargetAverageUtilization: &form.Cpu,
			},
		})
	}
	if form.Memory > 0 {
		metrics = append(metrics, v2beta1.MetricSpec{
			Type: v2beta1.ResourceMetricSourceType,
			Resource: &v2beta1.ResourceMetricSource{
				Name:                     v1.ResourceMemory,
				TargetAverageUtilization: &form.Memory,
			},
		})
	}

	in.Spec.Metrics = metrics
	in.Spec.MinReplicas = &form.MinReplicas
	in.Spec.MaxReplicas = form.MaxReplicas
	in.Spec.ScaleTargetRef = v2beta1.CrossVersionObjectReference{
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
