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

package apis

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	v1snap "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/storage"
	"kubegems.io/library/rest/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PvcHandler struct {
	C                client.Client
	PrometheusServer string
}

// @Tags			Agent.V1
// @Summary		获取PersistentVolumeClaim列表数据
// @Description	获取PersistentVolumeClaim列表数据
// @Accept			json
// @Produce		json
// @Param			order		query		string																					false	"page"
// @Param			search		query		string																					false	"search"
// @Param			page		query		int																						false	"page"
// @Param			size		query		int																						false	"page"
// @Param			namespace	path		string																					true	"namespace"
// @Param			cluster		path		string																					true	"cluster"
// @Success		200			{object}	handlers.ResponseStruct{Data=response.Page[v1.PersistentVolumeClaim]{List=[]object}}	"PersistentVolumeClaim"
// @Router			/v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pvcs [get]
// @Security		JWT
func (h *PvcHandler) List(c *gin.Context) {
	ns := c.Param("namespace")
	if ns == "_all" || ns == "_" {
		ns = ""
	}

	pvcList := &v1.PersistentVolumeClaimList{}
	listOpts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingLabelsSelector{Selector: getLabelSelector(c)},
	}
	if err := h.C.List(c.Request.Context(), pvcList, listOpts...); err != nil {
		NotOK(c, err)
		return
	}

	h.injectPVCRatioAnnotations(c, pvcList)

	// ignore errors
	pvcInUse, snapClassInUse, _ := h.getMapForSnapAndPod(c, ns)
	objects := pvcList.Items
	for _, obj := range objects {
		h.annotatePVC(c, &obj, pvcInUse, snapClassInUse)
	}
	pageData := response.PageObjectFromRequest(c.Request, objects)
	OK(c, pageData)
}

// @Tags			Agent.V1
// @Summary		获取PersistentVolumeClaim数据
// @Description	获取PersistentVolumeClaim数据
// @Accept			json
// @Produce		json
// @Param			cluster		path		string									true	"cluster"
// @Param			name		path		string									true	"name"
// @Param			namespace	path		string									true	"namespace"
// @Success		200			{object}	handlers.ResponseStruct{Data=object}	"counter"
// @Router			/v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pvcs/{name} [get]
// @Security		JWT
func (h *PvcHandler) Get(c *gin.Context) {
	ns := c.Param("namespace")
	pvcName := c.Param("name")

	pvc := &v1.PersistentVolumeClaim{}
	if ns == "_all" || ns == "_" {
		ns = ""
	}
	err := h.C.Get(c.Request.Context(), types.NamespacedName{Namespace: ns, Name: pvcName}, pvc)
	if err != nil {
		NotOK(c, err)
		return
	}

	pvcInUse, snapClassInUse, err := h.getMapForSnapAndPod(c, ns)
	if err != nil {
		NotOK(c, err)
		return
	}

	h.annotatePVC(c, pvc, pvcInUse, snapClassInUse)
	OK(c, pvc)
}

func (h *PvcHandler) annotatePVC(c *gin.Context, pvc *v1.PersistentVolumeClaim,
	pvcInUse map[string]int, snapClassInUse map[string]int,
) {
	inUse, allowSnapshot := false, false
	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}
	if _, ok := pvcInUse[pvc.Name]; ok {
		inUse = true
	}

	if provisioner := pvc.GetAnnotations()[storage.AnnotationStorageProvisioner]; len(provisioner) > 0 {
		if _, ok := snapClassInUse[provisioner]; ok {
			allowSnapshot = true
		}
	}

	pvc.Annotations[storage.AnnotationInUse] = strconv.FormatBool(inUse)
	pvc.Annotations[storage.AnnotationAllowSnapshot] = strconv.FormatBool(allowSnapshot)
}

func (h *PvcHandler) getMapForSnapAndPod(c *gin.Context, ns string) (map[string]int, map[string]int, error) {
	pvcInUse := make(map[string]int)
	snapClassInUse := make(map[string]int)

	podList := &v1.PodList{}
	err := h.C.List(c.Request.Context(), podList, client.InNamespace(ns))
	if err != nil {
		return nil, nil, err
	}

	for _, pod := range podList.Items {
		for _, pvc := range pod.Spec.Volumes {
			if pvc.PersistentVolumeClaim != nil {
				pvcInUse[pvc.PersistentVolumeClaim.ClaimName] = 1
			}
		}
	}

	snapClass := &v1snap.VolumeSnapshotClassList{}
	err = h.C.List(c.Request.Context(), snapClass, client.InNamespace(""))
	if err != nil {
		return nil, nil, err
	}

	for _, snap := range snapClass.Items {
		snapClassInUse[snap.Driver] = 1
	}

	return pvcInUse, snapClassInUse, nil
}

const (
	labelNamespace = "namespace"
	labelPvc       = "persistentvolumeclaim"
	querySort      = "sort"
)

var pvcRatio = []string{"ratio", "ratio-", "ratioDesc", "ratioAsc"}

// injectPVCRatioAnnotations
// 2025-07-15: 需求，要在前端PVC列表中支持按照使用率排序；
// 后端不支持直接排序，所以需要在后端获取所有PVC的使用率，附加在pvc的annotations中，在kubegems.io/library/rest/response中进行统一排序。
func (h *PvcHandler) injectPVCRatioAnnotations(c *gin.Context, pvcList *v1.PersistentVolumeClaimList) {
	if !slices.Contains(pvcRatio, c.Query(querySort)) {
		return
	}
	promClient, err := api.NewClient(api.Config{Address: h.PrometheusServer})
	if err != nil {
		return
	}

	v1api := promv1.NewAPI(promClient)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ret, _, err := v1api.Query(ctx, storage.MetricsPVCUsagePercent, time.Now())
	if err != nil {
		return
	}
	mapping := make(map[string]string)
	if val, ok := ret.(model.Vector); ok {
		for _, v := range val {
			value := v.Value.String()
			if value == "" {
				continue
			}
			namespace := string(v.Metric[labelNamespace])
			pvcName := string(v.Metric[labelPvc])
			if namespace == "" || pvcName == "" {
				continue
			}
			mapping[namespace+"/"+pvcName] = value
		}
	}
	for _, pvc := range pvcList.Items {
		if pvc.Annotations == nil {
			pvc.Annotations = make(map[string]string)
		}
		if ratio, ok := mapping[pvc.Namespace+"/"+pvc.Name]; ok {
			pvc.Annotations[storage.AnnotationStorageRatio] = ratio
		} else {
			pvc.Annotations[storage.AnnotationStorageRatio] = "0"
		}
	}
}
