package clusterhandler

import (
	"github.com/kubegems/gems/pkg/datas"
	"gorm.io/datatypes"
)

type ClusterQuota struct {
	Version        string                          `json:"version"`
	OversoldConfig datatypes.JSON                  `json:"oversoldConfig"`
	Resoruces      datas.ClusterResourceStatistics `json:"resources"`
	Workloads      datas.ClusterWorkloadStatistics `json:"workloads"`
}
