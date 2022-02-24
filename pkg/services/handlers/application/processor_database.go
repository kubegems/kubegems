package application

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
)

type DatabseProcessor struct {
	DB *gorm.DB
}

type EnvironmentDetails struct {
	ClusterName       string `json:"clusterName,omitempty"`
	ClusterKubeConfig []byte `json:"clusterKubeConfig,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
	EnvironmentName   string `json:"environmentName,omitempty"`
	EnvironmentType   string `json:"environmentType,omitempty"`
	// TenantName        string `json:"tenantName,omitempty"`
	// ProjectName       string `json:"projectName,omitempty"`
}

func (p *DatabseProcessor) GetEnvironmentWithCluster(ref PathRef) (*EnvironmentDetails, error) {
	env := &models.Environment{
		EnvironmentName: ref.Env,
		Project: &models.Project{
			ProjectName: ref.Project,
			Tenant: &models.Tenant{
				TenantName: ref.Tenant,
			},
		},
	}
	// if err := p.DB.Where(env).Preload("Cluster").Take(env).Error; err != nil {
	// 	return nil, err
	// }

	return &EnvironmentDetails{
		ClusterName:       env.Cluster.ClusterName,
		ClusterKubeConfig: env.Cluster.KubeConfig,
		Namespace:         env.Namespace,
		EnvironmentName:   env.EnvironmentName,
		EnvironmentType:   env.MetaType,
	}, nil
}

const (
	statusDeploy = 0b10
	statusApp    = 0b01
)

type syncStatus struct {
	deploy DeploiedManifest
	app    *models.Application
}

func (p *DatabseProcessor) RemoveDeploy(ctx context.Context, ref PathRef, deploy DeploiedManifest) error {
	exampleapp := &models.Application{
		ApplicationName: deploy.Name,
		Environment:     &models.Environment{EnvironmentName: ref.Env},
		Project: &models.Project{
			ProjectName: ref.Project,
			Tenant:      &models.Tenant{TenantName: ref.Tenant},
		},
	}

	exist := &models.Application{}
	if err := p.DB.Where(exampleapp).Delete(&exist).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		// remove existing
		return err
	}
	return nil
}

func (p *DatabseProcessor) SyncDeploy(ctx context.Context, ref PathRef, deploy DeploiedManifest) error {
	log := log.FromContextOrDiscard(ctx)

	if ref.IsEmpty() || deploy.Name == "" {
		return fmt.Errorf("invalid ref or deploy name")
	}

	exampleapp := &models.Application{
		ApplicationName: deploy.Name,
		Environment:     &models.Environment{EnvironmentName: ref.Env},
		Project: &models.Project{
			ProjectName: ref.Project,
			Tenant:      &models.Tenant{TenantName: ref.Tenant},
		},
	}

	exist := &models.Application{}
	if err := p.DB.Where(exampleapp).Take(&exist).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// create a new record
			updateDatabaseRecord(exampleapp, deploy)
			log.V(5).Info("create new record", "app", exampleapp)
			if err := p.DB.Create(exampleapp).Error; err != nil {
				return err
			}
		}
		return err
	}

	// update if necessary
	if updateDatabaseRecord(exist, deploy) {
		log.V(5).Info("update record", "app", exist)
		if err := p.DB.Save(exist).Error; err != nil {
			return err
		}
	}
	return nil
}

// SyncDeploies full sync current deploies to database
// Use git store as database instead of orm database on application, but sometimes database as data source from other component, so we need to sync it
func (p *DatabseProcessor) SyncDeploies(ctx context.Context, ref PathRef, deploies []DeploiedManifest) error {
	log := log.FromContextOrDiscard(ctx)

	depsmap := map[string]syncStatus{}
	for _, v := range deploies {
		depsmap[v.Name] = syncStatus{deploy: v}
	}

	// sync all apps
	apps := []models.Application{}
	if err := p.DB.Where(models.Application{
		Environment: &models.Environment{EnvironmentName: ref.Env},
		Project: &models.Project{
			ProjectName: ref.Project,
			Tenant:      &models.Tenant{TenantName: ref.Tenant},
		},
	}).Find(&apps).Error; err != nil {
		return err
	}

	for i, app := range apps {
		if dep, ok := depsmap[app.ApplicationName]; !ok {
			// remove existing
			log.Info("remove existing", "app", app)
			if err := p.DB.Delete(&app).Error; err != nil {
				return err
			}
		} else {
			// mark app existing
			dep.app = &apps[i]
		}
	}
	for _, dep := range depsmap {
		if dep.app != nil {
			// update existing
			if updateDatabaseRecord(dep.app, dep.deploy) {
				if err := p.DB.Save(dep.app).Error; err != nil {
					return err
				}
			}
		} else {
			// create app
			newapp := &models.Application{
				ApplicationName: dep.deploy.Name,
				Environment:     &models.Environment{EnvironmentName: ref.Env},
				Project:         &models.Project{ProjectName: ref.Project, Tenant: &models.Tenant{TenantName: ref.Tenant}},
			}
			updateDatabaseRecord(newapp, dep.deploy)
			if err := p.DB.Create(newapp).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func updateDatabaseRecord(app *models.Application, deploy DeploiedManifest) bool {
	updated := false
	if app.Kind != deploy.Kind {
		app.Kind = deploy.Kind
		updated = true
	}
	if app.Remark != deploy.Description {
		app.Remark = deploy.Description
		updated = true
	}
	if app.ApplicationName != deploy.Name {
		app.ApplicationName = deploy.Name
		updated = true
	}
	return updated
}
