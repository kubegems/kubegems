package apis

import (
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	gemlabels "kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/controller/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceHandler struct {
	C client.Client
}

var forbiddenBindNamespaces = []string{
	"kube-system",
	"istio-system",
	"gemcloud-gateway-system",
	"gemcloud-logging-system",
	"gemcloud-monitoring-system",
	"gemcloud-system",
	"gemcloud-workflow-system",
}

// @Tags Agent.V1
// @Summary 获取可以绑定的环境的namespace列表数据
// @Description 获取可以绑定的环境的namespace列表数据
// @Accept json
// @Produce json
// @Param order query string false "page"
// @Param search query string false "search"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param cluster path string true "cluster"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]object}} "Namespace"
// @Router /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces [get]
// @Security JWT
func (h *NamespaceHandler) List(c *gin.Context) {
	nsList := &corev1.NamespaceList{}
	sel := labels.NewSelector()
	req, _ := labels.NewRequirement(gemlabels.LabelEnvironment, selection.DoesNotExist, []string{})
	listOptions := &client.ListOptions{
		LabelSelector: sel.Add(*req),
	}
	if err := h.C.List(c.Request.Context(), nsList, listOptions); err != nil {
		NotOK(c, err)
		return
	}

	objects := []corev1.Namespace{}
	for _, obj := range nsList.Items {
		if !utils.StringIn(obj.Name, forbiddenBindNamespaces) {
			objects = append(objects, obj)
		}
	}
	pageData := NewPageDataFromContext(c, func(i int) SortAndSearchAble {
		return &objects[i]
	}, len(objects), objects)
	OK(c, pageData)
}
