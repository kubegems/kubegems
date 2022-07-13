package registryhandler

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/application"
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/harbor"
	"kubegems.io/kubegems/pkg/utils/slice"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	syncKindUpsert = "upsert"
	syncKindDelete = "delete"

	imagePullSecretKeyPrefix  = application.AnnotationImagePullSecretKeyPrefix
	defaultServiceAccountName = "default"
)

func (h *RegistryHandler) onChange(ctx context.Context, tx *gorm.DB, v *models.Registry) error {
	// validate
	if err := h.validate(ctx, v); err != nil {
		return err
	}
	// sync
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
					_, err := controllerutil.CreateOrUpdate(ctx, cli, secret, func() error {
						return updateSecretData(reg, secret)
					})
					if err != nil {
						return err
					}
				case syncKindDelete:
					if err := cli.Delete(ctx, secret); err != nil {
						return err
					}
				}

				// 默认仓库添加annotation
				updateEnviromentAnnotation(environment, defaultServiceAccountName, secretName, (kind == syncKindUpsert && reg.IsDefault))
				// 更新 env
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

func updateEnviromentAnnotation(env *v1beta1.Environment, serviceAccountName, targetSecretName string, isAdd bool) {
	if env.Annotations == nil {
		env.Annotations = make(map[string]string)
	}
	key := imagePullSecretKeyPrefix + serviceAccountName
	if _, exist := env.Annotations[key]; exist {
		secrets := strings.Split(env.Annotations[key], ",")
		if isAdd {
			if !slice.ContainStr(secrets, targetSecretName) {
				secrets = append(secrets, targetSecretName)
			}
		} else {
			secrets = slice.RemoveStrInReplace(secrets, targetSecretName)
		}
		if len(secrets) == 0 {
			delete(env.Annotations, key)
		} else {
			env.Annotations[key] = strings.Join(secrets, ",")
		}
	} else {
		if isAdd {
			env.Annotations[key] = targetSecretName
		}
	}
}
