package noproxy

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"kubegems.io/pkg/kubeclient"
	"kubegems.io/pkg/service/handlers"
)

type VolumeSnapshotRequest struct {
	// PersistentVolumeClaimName to snapshot
	PersistentVolumeClaimName string `binding:"required" json:"persistentVolumeClaimName,omitempty"`
	// Name of snapshot,如果不指定可自动生成。
	Name string `json:"name,omitempty"`
	// VolumeSnapshotClass 若未指定则使用与 pvc storageclass 同名的 snapshotcass
	VolumeSnapshotClass string `json:"volumeSnapshotClass,omitempty"`
}

// Snapshot 执行对PVC的快照
// @Tags NOPROXY
// @Summary 快照PVC
// @Description 执行对PVC的快照
// @Accept json
// @Produce json
// @Param cluster path string true "dev"
// @Param namespace path string true "default"
// @Param body body VolumeSnapshotRequest true "request body"
// @Success 200 {object} handlers.ResponseStruct{Data=v1.VolumeSnapshot} "VolumeSnapshotDefinition"
// @Failure 400 {object} handlers.ResponseStruct{} ""
// @Router /v1/noproxy/{cluster}/{namespace}/volumesnapshot [post]
// @Security JWT
func (vh *VolumeSnapshotHandler) Snapshot(c *gin.Context) {
	cluster := c.Params.ByName("cluster")
	namespace := c.Params.ByName("namespace")

	req := &VolumeSnapshotRequest{}
	if err := c.Bind(req); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if req.Name == "" {
		req.Name = req.PersistentVolumeClaimName + "-" + time.Now().Format("20060102150405")
	}

	vh.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	vh.SetAuditData(c, "创建", "持久卷快照", req.Name)

	pvc, err := kubeclient.GetClient().GetPersistentVolumeClaim(cluster, namespace, req.PersistentVolumeClaimName, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	var volumeSnapshotClassName *string
	storageclass, err := kubeclient.GetClient().GetStorageClass(cluster, *pvc.Spec.StorageClassName, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	snapshotclasses, err := kubeclient.GetClient().GetVolumeSnapshotClassList(cluster, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	for _, snapshotclass := range *snapshotclasses {
		if snapshotclass.Driver == storageclass.Provisioner {
			volumeSnapshotClassName = pointer.String(snapshotclass.Name)
		}
	}

	if volumeSnapshotClassName == nil {
		handlers.NotOK(c, fmt.Errorf("无法找到 pvc %s Provisioner=%s,的VolumeSnapshotClass", pvc.GetName(), storageclass.Provisioner))
		return
	}

	// todo(likun): replace with runtime.Encode()
	pvcbytes, err := json.Marshal(pvc)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	volumeSnapshot := &v1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
			Annotations: map[string]string{
				VolumeSnapshotAnnotationKeyPersistentVolumeClaim: string(pvcbytes),
			},
		},
		Spec: v1.VolumeSnapshotSpec{
			Source: v1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &req.PersistentVolumeClaimName,
			},
			VolumeSnapshotClassName: volumeSnapshotClassName,
		},
	}

	createdVolumeSnapshot, err := kubeclient.GetClient().CreateVolumeSnapshot(cluster, namespace, req.Name, volumeSnapshot)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, createdVolumeSnapshot)
}
