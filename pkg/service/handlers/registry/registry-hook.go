package registryhandler

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/service/handlers/registry/synchronizer"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/harbor"
)

func (h *RegistryHandler) onChange(ctx context.Context, tx *gorm.DB, v *models.Registry) error {
	// validate
	if err := h.validate(ctx, v); err != nil {
		return err
	}
	// sync
	if e := h.syncRegistry(ctx, v, tx, synchronizer.SyncKindUpsert); e != nil {
		return fmt.Errorf("同步镜像仓库信息到集群下失败 %w", e)
	}
	return nil
}

func (h *RegistryHandler) onDelete(ctx context.Context, tx *gorm.DB, v *models.Registry) error {
	if e := h.syncRegistry(ctx, v, tx, synchronizer.SyncKindDelete); e != nil {
		return fmt.Errorf("同步镜像仓库信息到集群下失败 %w", e)
	}
	return nil
}

const loginTimeout = 10 * time.Second

func (h *RegistryHandler) validate(ctx context.Context, v *models.Registry) error {
	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	// check if a harbor registry when enableExtends is true
	if v.EnableExtends {
		harborcli, err := harbor.NewClient(v.RegistryAddress, v.Username, v.Password)
		if err != nil {
			return err
		}
		systeminfo, err := harborcli.SystemInfo(ctx)
		if err != nil {
			return err
		}
		if systeminfo.HarborVersion == "" {
			return fmt.Errorf("can't get harbor version")
		}
	}
	// validate username/password
	if err := harbor.TryLogin(ctx, v.RegistryAddress, v.Username, v.Password); err != nil {
		return fmt.Errorf("try login registry: %w", err)
	}
	return nil
}

func (h *RegistryHandler) syncRegistry(ctx context.Context, reg *models.Registry, tx *gorm.DB, kind string) error {
	var envs []*models.Environment
	if e := tx.Preload("Cluster").Find(&envs, "project_id = ?", reg.ProjectID).Error; e != nil {
		return e
	}
	syncer := synchronizer.SynchronizerFor(h.BaseHandler)
	return syncer.SyncRegistries(ctx, envs, []*models.Registry{reg}, kind)
}
