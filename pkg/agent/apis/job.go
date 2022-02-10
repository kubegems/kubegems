package apis

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/agent/cluster"
	v1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type JobHandler struct {
	C       client.Client
	cluster cluster.Interface
}

// @Tags Agent.V1
// @Summary 获取Job列表数据
// @Description 获取Job列表数据
// @Accept json
// @Produce json
// @Param order query string false "page"
// @Param search query string false "search"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param namespace path string true "namespace"
// @Param cluster path string true "cluster"
// @Param topkind query string false "topkind(cronjob)"
// @Param topname query string false "topname"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]object}} "Job"
// @Router /v1/proxy/cluster/{cluster}/custom/batch/v1/namespaces/{namespace}/jobs [get]
// @Security JWT
func (h *JobHandler) List(c *gin.Context) {
	ns := c.Param("namespace")
	jobList := &v1.JobList{}
	if ns == "_all" || ns == "_" {
		ns = ""
	}

	listOptions := &client.ListOptions{
		Namespace:     ns,
		LabelSelector: getLabelSelector(c),
	}
	if err := h.C.List(c.Request.Context(), jobList, listOptions); err != nil {
		NotOK(c, err)
		return
	}

	objects := h.filterJobByTopname(c, jobList.Items)
	pageData := NewPageDataFromContext(c, func(i int) SortAndSearchAble {
		return &objects[i]
	}, len(objects), objects)

	if iswatch, _ := strconv.ParseBool(c.Query("watch")); iswatch {
		// list
		c.SSEvent("data", pageData)
		c.Writer.Flush()
		// watch
		WatchEvents(c, h.cluster, jobList, listOptions)
		return
	} else {
		OK(c, pageData)
	}
}

func (h *JobHandler) filterJobByTopname(c *gin.Context, jobs []v1.Job) []v1.Job {
	topkind := c.Query("topkind")
	topname := c.Query("topname")

	if len(topkind) == 0 || len(topname) == 0 {
		return jobs
	}
	var ret []v1.Job
	for _, job := range jobs {
		for _, owner := range job.OwnerReferences {
			if strings.EqualFold(owner.Kind, topkind) && owner.Name == topname {
				ret = append(ret, job)
			}
		}
	}
	return ret
}
