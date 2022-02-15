package forms

import "gorm.io/datatypes"

// +genform object:Cluster
type ClusterCommon struct {
	BaseForm
	ID          uint
	ClusterName string
	Primary     bool
	APIServer   string
	Version     string
	Runtime     string
}

// +genform object:Cluster
type ClusterDetail struct {
	BaseForm
	ID                   uint
	ClusterName          string
	APIServer            string
	KubeConfig           datatypes.JSON
	Version              string
	AgentAddr            string
	AgentCA              string
	AgentCert            string
	AgentKey             string
	Mode                 string
	Runtime              string
	Primary              bool
	OversoldConfig       datatypes.JSON
	Environments         []*EnvironmentCommon
	ClusterResourceQuota datatypes.JSON
}
