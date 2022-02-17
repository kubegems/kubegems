package registryhandler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/apis/application"
	"kubegems.io/pkg/apis/gems/v1beta1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/harbor"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	SearchFields   = []string{"RegistryName"}
	FilterFields   = []string{"RegistryName"}
	PreloadFields  = []string{"Creator", "Project"}
	OrderFields    = []string{"RegistryName", "ID"}
	ModelName      = "Registry"
	PrimaryKeyName = "registry_id"
)

// ListRegistry 列表 Registry
// @Tags Registry
// @Summary Registry列表
// @Description Registry列表
// @Accept json
// @Produce json
// @Param RegistryName query string false "RegistryName"
// @Param preload query string false "choices Creator,Project"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (RegistryName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Registry}} "Registry"
// @Router /v1/registry [get]
// @Security JWT
func (h *RegistryHandler) ListRegistry(c *gin.Context) {
	var list []models.Registry
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         ModelName,
		SearchFields:  SearchFields,
		PreloadFields: PreloadFields,
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// RetrieveRegistry Registry详情
// @Tags Registry
// @Summary Registry详情
// @Description get Registry详情
// @Accept json
// @Produce json
// @Param registry_id path uint true "registry_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "Registry"
// @Router /v1/registry/{registry_id} [get]
// @Security JWT
func (h *RegistryHandler) RetrieveRegistry(c *gin.Context) {
	var obj models.Registry
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PutRegistry 修改Registry
// @Tags Registry
// @Summary 修改Registry
// @Description 修改Registry
// @Accept json
// @Produce json
// @Param registry_id path uint true "registry_id"
// @Param param body models.Registry true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.Registry} "Registry"
// @Router /v1/registry/{registry_id} [put]
// @Security JWT
func (h *RegistryHandler) PutRegistry(c *gin.Context) {
	var obj models.Registry
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "修改", "镜像仓库", obj.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, obj.ProjectID)
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(obj.ID)) != c.Param(PrimaryKeyName) {
		handlers.NotOK(c, fmt.Errorf("请求体参数和URL参数ID不匹配"))
		return
	}

	// 检查其他默认仓库
	defaultRegistries := []models.Registry{}
	if err := h.GetDB().Where("project_id = ? and id != ? and is_default = ?", obj.ProjectID, obj.ID, true).
		Find(&defaultRegistries).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if len(defaultRegistries) > 0 && obj.IsDefault {
		handlers.NotOK(c, fmt.Errorf("默认仓库只能有一个"))
		return
	}

	ctx := c.Request.Context()
	// 检查用户名密码
	if err := h.validate(ctx, &obj); err != nil {
		handlers.NotOK(c, err)
		return
	}

	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&obj).Error; err != nil {
			return err
		}
		return h.onUpdate(ctx, tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}

	handlers.OK(c, obj)
}

// DeleteRegistry 删除 Registry
// @Tags Registry
// @Summary 删除 Registry
// @Description 删除 Registry
// @Accept json
// @Produce json
// @Param registry_id path uint true "registry_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/registry/{registry_id} [delete]
// @Security JWT
func (h *RegistryHandler) DeleteRegistry(c *gin.Context) {
	var obj models.Registry
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, err)
		return
	}
	h.SetAuditData(c, "删除", "镜像仓库", obj.RegistryName)
	h.SetExtraAuditData(c, models.ResProject, obj.ProjectID)

	ctx := c.Request.Context()

	err := h.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
			return err
		}
		return h.onDelete(ctx, tx, &obj)
	})
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.NoContent(c, nil)
}

const (
	syncKindUpsert = "upsert"
	syncKindDelete = "delete"

	imagePullSecretKeyPrefix  = application.AnnotationImagePullSecretKeyPrefix
	defaultServiceAccountName = "default"
)

func (h *RegistryHandler) validate(ctx context.Context, v *models.Registry) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
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

func (h *RegistryHandler) onUpdate(ctx context.Context, tx *gorm.DB, v *models.Registry) error {
	if e := h.syncRegistry(ctx, v, tx, syncKindUpsert); e != nil {
		return fmt.Errorf("同步镜像仓库信息到集群下失败 %w", e)
	}
	return nil
}

func (h *RegistryHandler) onDelete(ctx context.Context, tx *gorm.DB, v *models.Registry) error {
	if e := h.syncRegistry(ctx, v, tx, syncKindDelete); e != nil {
		return fmt.Errorf("同步镜像仓库信息到集群下失败 %w", e)
	}
	return nil
}

func (h *RegistryHandler) syncRegistry(ctx context.Context, reg *models.Registry, tx *gorm.DB, kind string) error {
	var envs []models.Environment
	if e := tx.Preload("Cluster").Find(&envs, "project_id = ?", reg.ProjectID).Error; e != nil {
		return e
	}

	secretName := reg.RegistryName

	// 并发处理env
	group := errgroup.Group{}
	for _, v := range envs {
		env := v // 必须重新赋值，ref. https://golang.org/doc/faq#closures_and_goroutines
		group.Go(func() error {
			return h.Execute(ctx, env.Cluster.ClusterName, func(ctx context.Context, cli agents.Client) error {
				environment := &v1beta1.Environment{}
				if err := cli.Get(ctx, client.ObjectKey{Name: env.EnvironmentName}, environment); err != nil {
					return err
				}

				secret := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: env.Namespace,
					},
				}
				switch kind {
				case syncKindUpsert:
					// 默认仓库添加annotation
					addOrRemoveSecret(environment, defaultServiceAccountName, secretName, reg.IsDefault)
					_, err := controllerutil.CreateOrUpdate(ctx, cli, secret, func() error {
						return updateSecretData(reg, secret)
					})
					return err
				case syncKindDelete:
					addOrRemoveSecret(environment, defaultServiceAccountName, secretName, false)
					return cli.Delete(ctx, secret)
				}
				return cli.Update(ctx, environment)
			})
		})
	}
	if err := group.Wait(); err != nil {
		log.Error(err, "sync registry")
		return err
	}
	return nil
}

func updateSecretData(v *models.Registry, in *v1.Secret) error {
	in.Type = v1.SecretTypeDockerConfigJson

	dockerAuthContent := map[string]interface{}{
		"auths": map[string]interface{}{
			v.RegistryAddress: map[string]interface{}{
				"username": v.Username,
				"password": v.Password,
				"email":    "",
				"auth":     base64.StdEncoding.EncodeToString([]byte(v.Username + ":" + v.Password)),
			},
		},
	}
	jsonStr, _ := json.Marshal(dockerAuthContent)
	if in.Data == nil {
		in.Data = make(map[string][]byte)
	}
	in.Data[v1.DockerConfigJsonKey] = jsonStr
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
