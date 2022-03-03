package orm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/harbor"
)

func (c *Client) registHook(obj client.Object, phase client.HookPhase, fn func(tx *gorm.DB, obj client.Object) error) {
	key := fmt.Sprintf("%s_%s", *obj.GetKind(), phase)
	c.hooks[key] = fn
}

func (c *Client) RegistHooks() {
	// 同步环境数据到集群
	c.registHook(&Environment{}, client.AfterUpdate, AfterEnvironmentCreateOrUpdate)
	c.registHook(&Environment{}, client.AfterCreate, AfterEnvironmentCreateOrUpdate)
	c.registHook(&Environment{}, client.AfterDelete, AfterEnvironmentDelete)
	// 同步删除租户资源
	c.registHook(&Tenant{}, client.AfterDelete, AfterTenantDelete)
	c.registHook(&TenantResourceQuota{}, client.AfterCreate, AfterTenantResourceQuotaCreateOrUpdate)
	c.registHook(&TenantResourceQuota{}, client.AfterUpdate, AfterTenantResourceQuotaCreateOrUpdate)

	// 同步删除项目下的环境资源
	c.registHook(&Project{}, client.AfterDelete, AfterProjectDelete)

	// 验证和同步同步镜像仓库的secrets
	c.registHook(&Registry{}, client.BeforeCreate, BeforeRegistryCreateOrUpdate)
	c.registHook(&Registry{}, client.BeforeUpdate, BeforeRegistryCreateOrUpdate)
	c.registHook(&Registry{}, client.AfterCreate, AfterRegistryCreateOrUpdate)
	c.registHook(&Registry{}, client.AfterUpdate, AfterRegistryCreateOrUpdate)
	c.registHook(&Registry{}, client.AfterDelete, AfterRegistryDelete)
}

func (c *Client) executeHook(obj client.Object, phase client.HookPhase) error {
	key := fmt.Sprintf("%s_%s", *obj.GetKind(), phase)
	hook, exist := c.hooks[key]
	if exist {
		return hook(c.db, obj)
	}
	return nil
}

func AfterEnvironmentDelete(tx *gorm.DB, obj client.Object) error {
	env := obj.(*Environment)
	return kubeClient(tx).DeleteEnvironment(env.Cluster.Name, env.Name)
}

func AfterEnvironmentCreateOrUpdate(tx *gorm.DB, obj client.Object) error {
	env := obj.(*Environment)
	var (
		project       Project
		cluster       Cluster
		spec          v1beta1.EnvironmentSpec
		tmpLimitRange map[string]v1.LimitRangeItem
		limitRange    []v1.LimitRangeItem
		resourceQuota v1.ResourceList
	)
	if e := tx.Preload("Tenant").First(&project, "id = ?", env.ProjectID).Error; e != nil {
		return e
	}
	if e := tx.First(&cluster, "id = ?", env.ClusterID).Error; e != nil {
		return e
	}

	if env.LimitRange != nil {
		e := json.Unmarshal(env.LimitRange, &tmpLimitRange)
		if e != nil {
			return e
		}
	}
	if env.ResourceQuota != nil {
		e := json.Unmarshal(env.ResourceQuota, &resourceQuota)
		if e != nil {
			return e
		}
	}

	for key, v := range tmpLimitRange {
		v.Type = v1.LimitType(key)
		limitRange = append(limitRange, v)
	}
	spec.Namespace = env.Namespace
	spec.Project = project.Name
	spec.Tenant = project.Tenant.Name
	spec.LimitRageName = "default"
	spec.ResourceQuotaName = "default"
	spec.DeletePolicy = env.DeletePolicy
	spec.ResourceQuota = resourceQuota
	if len(limitRange) > 0 {
		spec.LimitRage = limitRange
	}
	if e := kubeClient(tx).CreateOrUpdateEnvironment(cluster.Name, env.Name, spec); e != nil {
		return e
	}
	return nil
}

func AfterTenantDelete(tx *gorm.DB, obj client.Object) error {
	t := obj.(*Tenant)
	quotas := []TenantResourceQuota{}
	if e := tx.Preload("Cluster").Find(&quotas, TenantResourceQuota{TenantID: t.ID}).Error; e != nil {
		return e
	}
	for _, quota := range quotas {
		if err := kubeClient(tx).DeleteTenant(quota.Cluster.Name, t.Name); err != nil {
			log.Error(err, "delete crd tenant failed", "tenant", t.Name, "cluster", quota.Cluster.Name)
			return err
		}
	}
	return nil
}

func AfterTenantResourceQuotaDelete(tx *gorm.DB, obj client.Object) error {
	trq := obj.(*TenantResourceQuota)
	if err := kubeClient(tx).DeleteTenant(trq.Cluster.Name, trq.Tenant.Name); err != nil {
		return err
	}
	return nil
}

func AfterTenantResourceQuotaCreateOrUpdate(tx *gorm.DB, obj client.Object) error {
	trq := obj.(*TenantResourceQuota)
	var (
		tenant  Tenant
		cluster Cluster
		rels    []TenantUserRel
	)
	tx.First(&cluster, "id = ?", trq.ClusterID)
	tx.First(&tenant, "id = ?", trq.TenantID)
	tx.Preload("User").Find(&rels, "tenant_id = ?", trq.TenantID)

	admins := []string{}
	members := []string{}
	for _, rel := range rels {
		if rel.Role == TenantRoleAdmin {
			admins = append(admins, rel.User.Name)
		} else {
			members = append(members, rel.User.Name)
		}
	}
	// 创建or更新 租户
	if err := kubeClient(tx).CreateOrUpdateTenant(cluster.Name, tenant.Name, admins, members); err != nil {
		return err
	}
	// 这儿有个坑，controller还没有成功创建出来TenantResourceQuota，就去更新租户资源，会报错404；先睡会儿把
	<-time.NewTimer(time.Second * 2).C
	// 创建or更新 租户资源
	if err := kubeClient(tx).CreateOrUpdateTenantResourceQuota(cluster.Name, tenant.Name, trq.Content); err != nil {
		return err
	}
	return nil
}

func AfterProjectDelete(tx *gorm.DB, obj client.Object) error {
	project := obj.(*Project)
	environments := []Environment{}
	if err := tx.Preload("Cluster").Find(&environments, &Environment{ProjectID: project.ID}).Error; err != nil {
		return err
	}
	for _, env := range environments {
		e := kubeClient(tx).DeleteEnvironment(env.Cluster.Name, env.Name)
		if e != nil {
			return e
		}
	}
	// TODO: 删除 GIT 中的数据
	// TODO: 删除 ARGO 中的数据
	return nil
}

// Hooks

const (
	ResRegistry    = "registry"
	syncKindUpsert = "upsert"
	syncKindDelete = "delete"

	imagePullSecretKeyPrefix  = "kubegems.io/imagePullSecrets-"
	defaultServiceAccountName = "default"
)

func BeforeRegistryCreateOrUpdate(tx *gorm.DB, obj client.Object) error {
	v := obj.(*Registry)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := harbor.TryLogin(ctx, v.Address, v.Username, v.Password); err != nil {
		if err == context.DeadlineExceeded {
			return fmt.Errorf("验证用户名和密码超时")
		} else {
			return fmt.Errorf("验证用户名和密码错误 %v", err)
		}
	}
	return nil
}

func AfterRegistryCreateOrUpdate(tx *gorm.DB, obj client.Object) error {
	v := obj.(*Registry)
	if e := syncRegistry(tx, syncKindUpsert, v); e != nil {
		return fmt.Errorf("同步镜像仓库信息到集群下失败 %w", e)
	}
	return nil
}

func AfterRegistryDelete(tx *gorm.DB, obj client.Object) error {
	v := obj.(*Registry)
	if e := syncRegistry(tx, syncKindDelete, v); e != nil {
		return fmt.Errorf("同步镜像仓库信息到集群下失败 %w", e)
	}
	return nil
}

func syncRegistry(tx *gorm.DB, kind string, reg *Registry) error {
	var envs []Environment
	secretData := reg.buildSecretData()
	if e := tx.Preload("Cluster").Find(&envs, "project_id = ?", reg.ProjectID).Error; e != nil {
		return e
	}
	secretName := reg.Name

	group := errgroup.Group{}
	for idx := range envs {
		env := envs[idx]
		group.Go(func() error {
			envObj, e := kubeClient(tx).GetEnvironment(env.Cluster.Name, env.Name, nil)
			if e != nil {
				return e
			}

			if kind == syncKindUpsert {
				if e := kubeClient(tx).CreateOrUpdateSecret(env.Cluster.Name, env.Namespace, secretName, secretData); e != nil {
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
				if e := kubeClient(tx).DeleteSecretIfExist(env.Cluster.Name, env.Namespace, secretName); e != nil {
					return e
				}
				addOrRemoveSecret(envObj, defaultServiceAccountName, secretName, false)
			}

			if _, e := kubeClient(tx).PatchEnvironment(env.Cluster.Name, env.Name, envObj); e != nil {
				return e
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		log.Error(err, "同步镜像repo失败")
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
					secrets = utils.RemoveStr(secrets, targetSecretName)
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
			v.Address: map[string]interface{}{
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
