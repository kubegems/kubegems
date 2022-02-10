package noproxy

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/server/define"
)

type HpaHandler struct {
	define.ServerInterface
}

type PersistentVolumeClaimHandler struct {
	define.ServerInterface
}

type VolumeSnapshotHandler struct {
	define.ServerInterface
}

func (h *HpaHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.POST("/noproxy/:cluster/:namespace/hpa",
		h.CheckByClusterNamespace, h.SetObjectHpa)
	rg.GET("/noproxy/:cluster/:namespace/hpa", h.CheckByClusterNamespace, h.GetObjectHpa)
}

func (h *PersistentVolumeClaimHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.POST("/noproxy/:cluster/:namespace/persistentvolumeclaim",
		h.CheckByClusterNamespace, h.Create)
}

func (h *VolumeSnapshotHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.POST("/noproxy/:cluster/:namespace/volumesnapshot",
		h.CheckByClusterNamespace, h.Snapshot)
}
