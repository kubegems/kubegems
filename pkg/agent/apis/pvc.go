package apis

import (
	"strconv"

	"github.com/gin-gonic/gin"
	v1snap "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/apis/storage"
	"kubegems.io/kubegems/pkg/utils/pagination"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PvcHandler struct {
	C client.Client
}

// @Tags         Agent.V1
// @Summary      获取PersistentVolumeClaim列表数据
// @Description  获取PersistentVolumeClaim列表数据
// @Accept       json
// @Produce      json
// @Param        order      query     string                                                            false  "page"
// @Param        search     query     string                                                            false  "search"
// @Param        page       query     int                                                               false  "page"
// @Param        size       query     int                                                               false  "page"
// @Param        namespace  path      string                                                            true   "namespace"
// @Param        cluster    path      string                                                            true   "cluster"
// @Success      200        {object}  handlers.ResponseStruct{Data=pagination.PageData{List=[]object}}  "PersistentVolumeClaim"
// @Router       /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pvcs [get]
// @Security     JWT
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

	pvcInUse, snapClassInUse, err := h.getMapForSnapAndPod(c, ns)
	if err != nil {
		NotOK(c, err)
		return
	}

	objects := pvcList.Items
	for _, obj := range objects {
		h.annotatePVC(c, &obj, pvcInUse, snapClassInUse)
	}

	pageData := pagination.NewTypedSearchSortPageResourceFromContext(c, objects)
	OK(c, pageData)
}

// @Tags         Agent.V1
// @Summary      获取PersistentVolumeClaim数据
// @Description  获取PersistentVolumeClaim数据
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "cluster"
// @Param        name       path      string                                true  "name"
// @Param        namespace  path      string                                true  "namespace"
// @Success      200        {object}  handlers.ResponseStruct{Data=object}  "counter"
// @Router       /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/pvcs/{name} [get]
// @Security     JWT
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
