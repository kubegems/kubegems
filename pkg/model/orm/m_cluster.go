package orm

import "gorm.io/datatypes"

// +gen type:object kind:cluster pkcolume:id pkfield:ID preloads:TenantResourceQuotas
type Cluster struct {
	ID         uint   `gorm:"primarykey"`
	Name       string `gorm:"type:varchar(50);uniqueIndex"`
	APIServer  string `gorm:"type:varchar(250);uniqueIndex"`
	KubeConfig datatypes.JSON

	Version   string
	AgentAddr string
	AgentCA   string
	AgentCert string
	AgentKey  string
	Mode      string
	Runtime   string // docker or containerd
	Primary   bool   // is primary cluster

	OversoldConfig       datatypes.JSON // cluster oversold configuration
	Environments         []*Environment
	TenantResourceQuotas []*TenantResourceQuota
	ClusterResourceQuota datatypes.JSON
}
