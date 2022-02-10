package clusterhandler

import (
	"github.com/kubegems/gems/pkg/agent/apis/types"
	"gorm.io/datatypes"
)

type ClusterQuota struct {
	Version        string                          `json:"version"`
	OversoldConfig datatypes.JSON                  `json:"oversoldConfig"`
	Resoruces      types.ClusterResourceStatistics `json:"resources"`
	Workloads      types.ClusterWorkloadStatistics `json:"workloads"`
}
