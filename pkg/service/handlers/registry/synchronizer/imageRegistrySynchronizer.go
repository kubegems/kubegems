package synchronizer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/application"
	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/set"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	SyncKindUpsert = "upsert"
	SyncKindDelete = "delete"

	imagePullSecretKeyPrefix  = application.AnnotationImagePullSecretKeyPrefix
	defaultServiceAccountName = "default"
)

type ImageRegistrySynchronizer struct {
	base.BaseHandler
}

func SynchronizerFor(h base.BaseHandler) *ImageRegistrySynchronizer {
	return &ImageRegistrySynchronizer{
		BaseHandler: h,
	}
}

func (h *ImageRegistrySynchronizer) SyncRegistries(ctx context.Context, environments []*models.Environment, registries []*models.Registry, kind string) error {

	for _, env := range environments {
		if env.Cluster == nil {
			return errors.New("failed to SyncRegistries, required environments with cluster property")
		}
	}

	group := errgroup.Group{}
	for idx := range environments {
		env := environments[idx]
		group.Go(func() error {
			cli, err := h.GetAgents().ClientOf(ctx, env.Cluster.ClusterName)
			if err != nil {
				return err
			}
			environment := &v1beta1.Environment{}
			if err := cli.Get(ctx, client.ObjectKey{Name: env.EnvironmentName}, environment); err != nil {
				return err
			}
			secrets := []string{}
			// TODO: Is it necessary to update/delete secrets concurrenctly ??
			for _, reg := range registries {
				secretName := reg.RegistryName
				secret := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: env.Namespace,
					},
				}
				switch kind {
				case SyncKindUpsert:
					_, err := controllerutil.CreateOrUpdate(ctx, cli, secret, func() error {
						return updateSecretData(reg, secret)
					})
					if err != nil {
						return err
					}
					if reg.IsDefault {
						secrets = append(secrets, secretName)
					}
				case SyncKindDelete:
					if err := cli.Delete(ctx, secret); err != nil {
						return err
					}
					secrets = append(secrets, secretName)
				}
			}

			_, err = controllerutil.CreateOrUpdate(ctx, cli, environment, func() error {
				UpdateEnviromentAnnotation(environment, defaultServiceAccountName, secrets, kind == SyncKindUpsert)
				return nil
			})
			return err
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

func UpdateEnviromentAnnotation(env *v1beta1.Environment, serviceAccountName string, newSecrets []string, isAdd bool) {
	if env.Annotations == nil {
		env.Annotations = make(map[string]string)
	}
	key := imagePullSecretKeyPrefix + serviceAccountName
	if _, exist := env.Annotations[key]; exist {
		secrets := strings.Split(env.Annotations[key], ",")

		secretSet := set.NewSet[string]()
		if isAdd {
			secretSet.Append(secrets...)
			secretSet.Append(newSecrets...)
		} else {
			secretSet.Remove(newSecrets...)
		}
		secrets = secretSet.Slice()
		if len(secrets) == 0 {
			delete(env.Annotations, key)
		} else {
			env.Annotations[key] = strings.Join(secrets, ",")
		}
	} else {
		if isAdd {
			env.Annotations[key] = strings.Join(newSecrets, ",")
		}
	}
}
