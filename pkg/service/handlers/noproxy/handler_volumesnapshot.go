package noproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"kubegems.io/kubegems/pkg/apis/storage"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/agents"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
// @Tags         NOPROXY
// @Summary      快照PVC
// @Description  执行对PVC的快照
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                           true  "dev"
// @Param        namespace  path      string                                           true  "default"
// @Param        body       body      VolumeSnapshotRequest                            true  "request body"
// @Success      200        {object}  handlers.ResponseStruct{Data=v1.VolumeSnapshot}  "VolumeSnapshotDefinition"
// @Failure      400        {object}  handlers.ResponseStruct{}                        ""
// @Router       /v1/noproxy/{cluster}/{namespace}/volumesnapshot [post]
// @Security     JWT
func (vh *VolumeSnapshotHandler) Snapshot(c *gin.Context) {
	cluster := c.Params.ByName("cluster")
	namespace := c.Params.ByName("namespace")

	req := &VolumeSnapshotRequest{}
	if err := c.Bind(req); err != nil {
		handlers.NotOK(c, err)
		return
	}
	ctx := c.Request.Context()

	if req.Name == "" {
		req.Name = req.PersistentVolumeClaimName + "-" + time.Now().Format("20060102150405")
	}

	vh.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	vh.SetAuditData(c, "创建", "持久卷快照", req.Name)

	err := vh.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.PersistentVolumeClaimName,
				Namespace: namespace,
			},
		}
		if err := cli.Get(ctx, client.ObjectKeyFromObject(pvc), pvc); err != nil {
			return err
		}

		storageclass := &v1beta1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: *pvc.Spec.StorageClassName,
			},
		}

		if err := cli.Get(ctx, client.ObjectKeyFromObject(storageclass), storageclass); err != nil {
			return err
		}
		snapshotclasses := &v1.VolumeSnapshotClassList{}
		if err := cli.List(ctx, snapshotclasses); err != nil {
			return err
		}

		var volumeSnapshotClassName *string
		for _, snapshotclass := range snapshotclasses.Items {
			if snapshotclass.Driver == storageclass.Provisioner {
				volumeSnapshotClassName = pointer.String(snapshotclass.Name)
			}
		}
		if volumeSnapshotClassName == nil {
			return fmt.Errorf("unable to find VolumeSnapshotClass of pvc %s Provisioner=%s", pvc.GetName(), storageclass.Provisioner)
		}

		pvcbytes, err := json.Marshal(pvc)
		if err != nil {
			return err
		}

		volumeSnapshot := &v1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Name,
				Annotations: map[string]string{
					storage.AnnotationVolumeSnapshotAnnotationKeyPersistentVolumeClaim: string(pvcbytes),
				},
				Namespace: namespace,
			},
			Spec: v1.VolumeSnapshotSpec{
				Source: v1.VolumeSnapshotSource{
					PersistentVolumeClaimName: &req.PersistentVolumeClaimName,
				},
				VolumeSnapshotClassName: volumeSnapshotClassName,
			},
		}

		return cli.Create(ctx, volumeSnapshot)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}
