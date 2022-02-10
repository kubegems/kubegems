package apis

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MetricsHandler struct {
	C                 client.Client
	metricScraperHost string
}

// @Tags Agent.V1
// @Summary 获取Node实时Metrics
// @Description 获取Nodo实时Metrics
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Success 200 {object} handlers.ResponseStruct{Data=[]v1beta1.NodeMetrics} "metrics"
// @Router /v1/proxy/cluster/{cluster}/custom/metrics.k8s.io/v1beta1/nodes [get]
// @Security JWT
func (h *MetricsHandler) Nodes(c *gin.Context) {
	nodeMetricsList := &v1beta1.NodeMetricsList{}
	if err := h.C.List(c.Request.Context(), nodeMetricsList); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, nodeMetricsList.Items)
}

// @Tags Agent.V1
// @Summary 获取指定Node实时Metrics
// @Description 获取指定Nodo实时Metrics
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=v1beta1.NodeMetrics} "metrics"
// @Router /v1/proxy/cluster/{cluster}/custom/metrics.k8s.io/v1beta1/nodes/{name} [get]
// @Security JWT
func (h *MetricsHandler) Node(c *gin.Context) {
	nodeMetrics := &v1beta1.NodeMetrics{}
	err := h.C.Get(c.Request.Context(), types.NamespacedName{Name: c.Param("name")}, nodeMetrics)
	if err != nil {
		NotOK(c, err)
		return
	}
	OK(c, nodeMetrics)
}

// @Tags Agent.V1
// @Summary 获取Pods实时Metrics
// @Description 获取Pods实时Metrics
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Success 200 {object} handlers.ResponseStruct{Data=[]v1beta1.PodMetrics} "metrics"
// @Router /v1/proxy/cluster/{cluster}/custom/metrics.k8s.io/v1beta1/namespaces/{namespace}/pods} [get]
// @Security JWT
// Deprecated _all namespace 在路径解析时统一处理
func (h *MetricsHandler) Pods(c *gin.Context) {
	namespace := c.Param("namespace")
	ctx := c.Request.Context()
	var (
		podsMetrics *v1beta1.PodMetricsList
		err         error
	)
	podMetrics := &v1beta1.PodMetricsList{}
	if namespace == "_all" {
		err = h.C.List(ctx, podMetrics)
	} else {
		err = h.C.List(ctx, podMetrics, client.InNamespace(namespace))
	}
	if err != nil {
		NotOK(c, err)
		return
	}
	OK(c, podsMetrics)
}

// @Tags Agent.V1
// @Summary 获取Nodes最近十五分钟的Metrics(从scraper获取)
// @Description 获取Nodes最近十五分钟的Metrics(从scraper获取)
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Success 200 {object} handlers.ResponseStruct{Data=[]object} "metrics"
// @Router /v1/proxy/cluster/{cluster}/custom/metrics.k8s.io/v1beta1/nodes/actions/recently [get]
// @Security JWT
func (mh *MetricsHandler) NodeList(c *gin.Context) {
	client := resty.New()
	req := client.R()
	uri := fmt.Sprintf("%s%s", mh.metricScraperHost, "/api/v2/dashboard/nodes/metrics")
	resp, err := req.Get(uri)
	if err != nil {
		NotOK(c, err)
		return
	}
	r := map[string]interface{}{}
	_ = json.Unmarshal(resp.Body(), &r)
	OK(c, r)
}

// @Tags Agent.V1
// @Summary 获取Pods最近十五分钟的Metrics(从scraper获取)
// @Description 获取Pods最近十五分钟的Metrics(从scraper获取)
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param pods query string false "pods"
// @Success 200 {object} handlers.ResponseStruct{Data=[]object} "metrics"
// @Router /v1/proxy/cluster/{cluster}/custom/metrics.k8s.io/v1beta1/namespaces/{namespace}/pods/actions/recently [get]
// @Security JWT
func (mh *MetricsHandler) PodList(c *gin.Context) {
	client := resty.New()
	req := client.R()
	q := url.Values{}
	q.Add("pods", c.Query("pods"))
	uri := fmt.Sprintf("%s%s", mh.metricScraperHost, "/api/v2/dashboard/pods/metrics?"+q.Encode())
	resp, err := req.Get(uri)
	if err != nil {
		NotOK(c, err)
		return
	}
	r := map[string]interface{}{}
	_ = json.Unmarshal(resp.Body(), &r)
	OK(c, r)
}
