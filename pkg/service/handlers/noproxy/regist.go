package noproxy

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type HpaHandler struct {
	base.BaseHandler
}

type PersistentVolumeClaimHandler struct {
	base.BaseHandler
}

type VolumeSnapshotHandler struct {
	base.BaseHandler
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
