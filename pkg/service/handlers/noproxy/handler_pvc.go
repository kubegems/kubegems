package noproxy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/apis/storage"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/agents"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PersistentVolumeClaimRequest struct {
	Name               string `json:"name,omitempty"`               // Name 需要恢复到的pvc名称
	VolumeSnapshotName string `json:"volumeSnapshotName,omitempty"` // VolumeSnapshotName 需要被恢复的快照
}

// Create 恢复卷快照到新pvc
// @Tags NOPROXY
// @Summary 从快照恢复PVC
// @Description 从快照恢复PVC
// @Accept json
// @Produce json
// @Param cluster path string true "dev"
// @Param namespace path string true "default"
// @Param body body PersistentVolumeClaimRequest true "request body"
// @Success 200 {object} handlers.ResponseStruct{Data=v1.PersistentVolumeClaim} "PersistentVolumeClaim"
// @Failure 400 {object} handlers.ResponseStruct{} ""
// @Router /v1/noproxy/{cluster}/{namespace}/persistentvolumeclaim [post]
// @Security JWT
func (h *PersistentVolumeClaimHandler) Create(c *gin.Context) {
	cluster := c.Params.ByName("cluster")
	namespace := c.Params.ByName("namespace")
	req := &PersistentVolumeClaimRequest{}
	if err := c.Bind(req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "恢复", "快照到持久卷", req.Name)

	volumeSnapshotName := req.VolumeSnapshotName

	ctx := c.Request.Context()

	err := h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		volumesnapshot := &snapv1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      volumeSnapshotName,
				Namespace: namespace,
			},
		}

		if err := cli.Get(ctx, client.ObjectKeyFromObject(volumesnapshot), volumesnapshot); err != nil {
			return err
		}
		if volumesnapshot.Status.ReadyToUse != nil && !*volumesnapshot.Status.ReadyToUse {
			return fmt.Errorf("volumesnapshot %v status is %v", volumeSnapshotName, volumesnapshot.Status.ReadyToUse)
		}

		pvcbytes := volumesnapshot.Annotations[storage.AnnotationVolumeSnapshotAnnotationKeyPersistentVolumeClaim]

		pvc := &v1.PersistentVolumeClaim{}
		if err := json.Unmarshal([]byte(pvcbytes), pvc); err != nil {
			return err
		}

		pvc.DeletionTimestamp = nil
		pvc.Name = req.Name
		pvc.Namespace = namespace
		pvc.ResourceVersion = ""
		pvc.Annotations = map[string]string{}
		group := snapv1.GroupName
		pvc.Spec.DataSource = &v1.TypedLocalObjectReference{
			APIGroup: &group,
			Kind:     "VolumeSnapshot",
			Name:     volumesnapshot.Name,
		}
		// reset bind volume
		pvc.Spec.VolumeName = ""
		pvc.Status = v1.PersistentVolumeClaimStatus{}

		return cli.Create(ctx, pvc)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}
