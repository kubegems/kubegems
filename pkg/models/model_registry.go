package models

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/harbor"
)

const (
	ResRegistry    = "registry"
	syncKindUpsert = "upsert"
	syncKindDelete = "delete"

	imagePullSecretKeyPrefix  = "gems.cloudminds.com/imagePullSecrets-"
	defaultServiceAccountName = "default"
)

// Registry 镜像仓库表
type Registry struct {
	ID uint `gorm:"primarykey"`
	// 仓库名称
	RegistryName string `gorm:"type:varchar(50);uniqueIndex:uniq_idx_project_registry;"`
	// 仓库地址
	RegistryAddress string `gorm:"type:varchar(512)"`
	// 用户名
	Username string `gorm:"type:varchar(50)"`
	// 密码
	Password string `gorm:"type:varchar(512)"`
	// 创建者
	Creator *User
	// 更新时间
	UpdateTime time.Time
	CreatorID  uint
	Project    *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 项目ID
	ProjectID uint `grom:"uniqueIndex:uniq_idx_project_registry;"`
	IsDefault bool
}

type NoPasswordRegistry struct {
	ID uint `gorm:"primarykey"`
	// 仓库名称
	RegistryName string `gorm:"type:varchar(50);uniqueIndex"`
	// 仓库地址
	RegistryAddress string `gorm:"type:varchar(512)"`
	// 用户名
	Username string `gorm:"type:varchar(50)"`
	// 创建者
	Creator *User
	// 更新时间
	UpdateTime time.Time
	CreatorID  uint
	Project    *Project `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	// 项目ID
	ProjectID uint
}

func (v *Registry) BeforeSave(tx *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := harbor.TryLogin(ctx, v.RegistryAddress, v.Username, v.Password); err != nil {
		if err == context.DeadlineExceeded {
			return fmt.Errorf("验证用户名和密码超时")
		} else {
			return fmt.Errorf("验证用户名和密码错误 %w", err)
		}
	}
	return nil
}

func (v *Registry) AfterSave(tx *gorm.DB) error {
	if e := v.syncRegistry(tx, syncKindUpsert); e != nil {
		return fmt.Errorf("同步镜像仓库信息到集群下失败 %w", e)
	}
	return nil
}

func (v *Registry) AfterDelete(tx *gorm.DB) error {
	if e := v.syncRegistry(tx, syncKindDelete); e != nil {
		return fmt.Errorf("同步镜像仓库信息到集群下失败 %w", e)
	}
	return nil
}

func (reg *Registry) syncRegistry(tx *gorm.DB, kind string) error {
	var envs []Environment
	secretData := reg.buildSecretData()
	if e := tx.Preload("Cluster").Find(&envs, "project_id = ?", reg.ProjectID).Error; e != nil {
		return e
	}
	secretName := reg.RegistryName

	// 并发处理env
	group := errgroup.Group{}
	for _, v := range envs {
		env := v // 必须重新赋值，ref. https://golang.org/doc/faq#closures_and_goroutines
		group.Go(func() error {
			envObj, e := GetKubeClient().GetEnvironment(env.Cluster.ClusterName, env.EnvironmentName, nil)
			if e != nil {
				return e
			}

			if kind == syncKindUpsert {
				if e := GetKubeClient().CreateOrUpdateSecret(env.Cluster.ClusterName, env.Namespace, secretName, secretData); e != nil {
					return e
				}
				// 默认仓库添加annotation
				if reg.IsDefault {
					addOrRemoveSecret(envObj, defaultServiceAccountName, secretName, true)
				} else {
					addOrRemoveSecret(envObj, defaultServiceAccountName, secretName, false)
				}
			}
			if kind == syncKindDelete {
				if e := GetKubeClient().DeleteSecretIfExist(env.Cluster.ClusterName, env.Namespace, secretName); e != nil {
					return e
				}
				addOrRemoveSecret(envObj, defaultServiceAccountName, secretName, false)
			}

			if _, e := GetKubeClient().PatchEnvironment(env.Cluster.ClusterName, env.EnvironmentName, envObj); e != nil {
				return e
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		log.Error(err, "sync registry")
		return err
	}
	return nil
}

func addOrRemoveSecret(env *v1beta1.Environment, serviceAccountName, targetSecretName string, isAdd bool) {
	if env.Annotations == nil {
		env.Annotations = make(map[string]string)
	}

	if len(env.Annotations) == 0 && isAdd {
		env.Annotations = map[string]string{
			imagePullSecretKeyPrefix + serviceAccountName: targetSecretName,
		}
		return
	}

	for k := range env.Annotations {
		if strings.HasPrefix(k, imagePullSecretKeyPrefix) {
			saName := strings.TrimPrefix(k, imagePullSecretKeyPrefix)
			if saName == serviceAccountName {
				secrets := strings.Split(env.Annotations[k], ",")
				if isAdd {
					if !utils.ContainStr(secrets, targetSecretName) {
						secrets = append(secrets, targetSecretName)
					}
				} else {
					secrets = utils.RemoveStrInReplace(secrets, targetSecretName)
				}

				if len(secrets) == 0 {
					env.Annotations = nil
				} else {
					env.Annotations[k] = strings.Join(secrets, ",")
				}
			}
		}
	}
}

func (v *Registry) buildSecretData() map[string][]byte {
	authStr := base64.StdEncoding.EncodeToString([]byte(v.Username + ":" + v.Password))
	dockerAuthContent := map[string]interface{}{
		"auths": map[string]interface{}{
			v.RegistryAddress: map[string]interface{}{
				"username": v.Username,
				"password": v.Password,
				"email":    "",
				"auth":     authStr,
			},
		},
	}
	jsonStr, _ := json.Marshal(dockerAuthContent)
	return map[string][]byte{
		".dockerconfigjson": jsonStr,
	}
}
