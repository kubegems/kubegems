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

package registryhandler

import (
	"context"
	"errors"
	"net"
	"regexp"
	"time"

	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/i18n"
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
		return i18n.Errorf(ctx, "Failed to synchronize the image registry information to the cluster: %w", e)
	}
	return nil
}

func (h *RegistryHandler) onDelete(ctx context.Context, tx *gorm.DB, v *models.Registry) error {
	if e := h.syncRegistry(ctx, v, tx, synchronizer.SyncKindDelete); e != nil {
		return i18n.Errorf(ctx, "Failed to synchronize the image registry information to the cluster: %w", e)
	}
	return nil
}

const loginTimeout = 10 * time.Second

func (h *RegistryHandler) validate(ctx context.Context, v *models.Registry) error {
	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	if err := ValidateRegistryAddress(ctx, v.RegistryAddress); err != nil {
		return err
	}

	// check if a harbor registry when enableExtends is true
	if v.EnableExtends {
		harborcli := harbor.NewClient(v.RegistryAddress, v.Username, v.Password)
		systeminfo, err := harborcli.SystemInfo(ctx)
		if err != nil {
			return err
		}
		if systeminfo.HarborVersion == "" {
			return i18n.Errorf(ctx, "failed to get Harbor version")
		}
	}
	// validate username/password
	if err := harbor.TryLogin(ctx, v.RegistryAddress, v.Username, v.Password); err != nil {
		neterr := &net.OpError{}
		if errors.As(err, &neterr) {
			return i18n.Errorf(ctx, "failed to connect to the registry address: %s", err.Error())
		}
		return i18n.Errorf(ctx, "invalid username or password")
	}
	return nil
}

// domain regexp
var domainRegexp = regexp.MustCompile(`^https?://([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.[a-zA-Z]{2,}$`)

func ValidateRegistryAddress(ctx context.Context, address string) error {
	if address == "" {
		return i18n.Errorf(ctx, "registry address is required")
	}
	if !domainRegexp.MatchString(address) {
		return i18n.Errorf(ctx, "invalid registry address, must be a valid domain name matching the pattern: %s", domainRegexp.String())
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
