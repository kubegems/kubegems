package application

import (
	"gorm.io/gorm"
	"kubegems.io/pkg/models"
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
	if err := p.DB.Where(env).Preload("Cluster").Take(env).Error; err != nil {
		return nil, err
	}

	return &EnvironmentDetails{
		ClusterName:       env.Cluster.ClusterName,
		ClusterKubeConfig: env.Cluster.KubeConfig,
		Namespace:         env.Namespace,
		EnvironmentName:   env.EnvironmentName,
		EnvironmentType:   env.MetaType,
	}, nil
}
