package forms

import "gorm.io/datatypes"

// +genform object:Cluster
type ClusterCommon struct {
	BaseForm
	ID        uint   `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Primary   bool   `json:"primary,omitempty"`
	APIServer string `json:"apiServer,omitempty"`
	Version   string `json:"version,omitempty"`
	Runtime   string `json:"runtime,omitempty"`
}

// +genform object:Cluster
type ClusterDetail struct {
	BaseForm
	ID                   uint                 `json:"id,omitempty"`
	Name                 string               `json:"name,omitempty"`
	APIServer            string               `json:"apiServer,omitempty"`
	KubeConfig           datatypes.JSON       `json:"kubeConfig,omitempty"`
	Version              string               `json:"version,omitempty"`
	AgentAddr            string               `json:"agentAddr,omitempty"`
	AgentCA              string               `json:"agentCA,omitempty"`
	AgentCert            string               `json:"agentCert,omitempty"`
	AgentKey             string               `json:"agentKey,omitempty"`
	Mode                 string               `json:"mode,omitempty"`
	Runtime              string               `json:"runtime,omitempty"`
	Primary              bool                 `json:"primary,omitempty"`
	OversoldConfig       datatypes.JSON       `json:"oversoldConfig,omitempty"`
	Environments         []*EnvironmentCommon `json:"environments,omitempty"`
	ClusterResourceQuota datatypes.JSON       `json:"clusterResourceQuota,omitempty"`
}
