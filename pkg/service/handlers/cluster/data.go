package clusterhandler

import (
	"gorm.io/datatypes"
	"kubegems.io/pkg/agent/apis/types"
)

type ClusterQuota struct {
	Version        string                          `json:"version"`
	OversoldConfig datatypes.JSON                  `json:"oversoldConfig"`
	Resoruces      types.ClusterResourceStatistics `json:"resources"`
	Workloads      types.ClusterWorkloadStatistics `json:"workloads"`
}
