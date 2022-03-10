package orm

var (
	applicationKind          = "application"
	applicationTable         = "applications"
	applicationPrimaryKey    = "id"
	applicationValidPreloads = []string{"Environment"}
)

func (obj *Application) TableName() *string {
	return &applicationTable
}

func (obj *Application) GetKind() *string {
	return &applicationKind
}

func (obj *Application) PrimaryKeyField() *string {
	return &applicationPrimaryKey
}

func (obj *Application) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Application) PreloadFields() *[]string {
	return &applicationValidPreloads
}

type ApplicationList struct {
	Items []*Application
	BaseList
}

func (objList *ApplicationList) GetKind() *string {
	return &applicationKind
}

func (obj *ApplicationList) PrimaryKeyField() *string {
	return &applicationPrimaryKey
}

func (objList *ApplicationList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *ApplicationList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *ApplicationList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *ApplicationList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *ApplicationList) DataPtr() interface{} {
	return &objList.Items
}

var (
	auditLogKind          = "audit_log"
	auditLogTable         = "audit_logs"
	auditLogPrimaryKey    = "id"
	auditLogValidPreloads = []string{}
)

func (obj *AuditLog) TableName() *string {
	return &auditLogTable
}

func (obj *AuditLog) GetKind() *string {
	return &auditLogKind
}

func (obj *AuditLog) PrimaryKeyField() *string {
	return &auditLogPrimaryKey
}

func (obj *AuditLog) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *AuditLog) PreloadFields() *[]string {
	return &auditLogValidPreloads
}

type AuditLogList struct {
	Items []*AuditLog
	BaseList
}

func (objList *AuditLogList) GetKind() *string {
	return &auditLogKind
}

func (obj *AuditLogList) PrimaryKeyField() *string {
	return &auditLogPrimaryKey
}

func (objList *AuditLogList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *AuditLogList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *AuditLogList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *AuditLogList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *AuditLogList) DataPtr() interface{} {
	return &objList.Items
}

var (
	authSourceKind          = "auth_source"
	authSourceTable         = "auth_sources"
	authSourcePrimaryKey    = "id"
	authSourceValidPreloads = []string{}
)

func (obj *AuthSource) TableName() *string {
	return &authSourceTable
}

func (obj *AuthSource) GetKind() *string {
	return &authSourceKind
}

func (obj *AuthSource) PrimaryKeyField() *string {
	return &authSourcePrimaryKey
}

func (obj *AuthSource) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *AuthSource) PreloadFields() *[]string {
	return &authSourceValidPreloads
}

type AuthSourceList struct {
	Items []*AuthSource
	BaseList
}

func (objList *AuthSourceList) GetKind() *string {
	return &authSourceKind
}

func (obj *AuthSourceList) PrimaryKeyField() *string {
	return &authSourcePrimaryKey
}

func (objList *AuthSourceList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *AuthSourceList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *AuthSourceList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *AuthSourceList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *AuthSourceList) DataPtr() interface{} {
	return &objList.Items
}

var (
	chartRepoKind          = "chart_repo"
	chartRepoTable         = "chart_repos"
	chartRepoPrimaryKey    = "id"
	chartRepoValidPreloads = []string{}
)

func (obj *ChartRepo) TableName() *string {
	return &chartRepoTable
}

func (obj *ChartRepo) GetKind() *string {
	return &chartRepoKind
}

func (obj *ChartRepo) PrimaryKeyField() *string {
	return &chartRepoPrimaryKey
}

func (obj *ChartRepo) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *ChartRepo) PreloadFields() *[]string {
	return &chartRepoValidPreloads
}

type ChartRepoList struct {
	Items []*ChartRepo
	BaseList
}

func (objList *ChartRepoList) GetKind() *string {
	return &chartRepoKind
}

func (obj *ChartRepoList) PrimaryKeyField() *string {
	return &chartRepoPrimaryKey
}

func (objList *ChartRepoList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *ChartRepoList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *ChartRepoList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *ChartRepoList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *ChartRepoList) DataPtr() interface{} {
	return &objList.Items
}

var (
	clusterKind          = "cluster"
	clusterTable         = "clusters"
	clusterPrimaryKey    = "id"
	clusterValidPreloads = []string{"TenantResourceQuotas"}
)

func (obj *Cluster) TableName() *string {
	return &clusterTable
}

func (obj *Cluster) GetKind() *string {
	return &clusterKind
}

func (obj *Cluster) PrimaryKeyField() *string {
	return &clusterPrimaryKey
}

func (obj *Cluster) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Cluster) PreloadFields() *[]string {
	return &clusterValidPreloads
}

type ClusterList struct {
	Items []*Cluster
	BaseList
}

func (objList *ClusterList) GetKind() *string {
	return &clusterKind
}

func (obj *ClusterList) PrimaryKeyField() *string {
	return &clusterPrimaryKey
}

func (objList *ClusterList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *ClusterList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *ClusterList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *ClusterList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *ClusterList) DataPtr() interface{} {
	return &objList.Items
}

var (
	containerKind          = "container"
	containerTable         = "containers"
	containerPrimaryKey    = "id"
	containerValidPreloads = []string{}
)

func (obj *Container) TableName() *string {
	return &containerTable
}

func (obj *Container) GetKind() *string {
	return &containerKind
}

func (obj *Container) PrimaryKeyField() *string {
	return &containerPrimaryKey
}

func (obj *Container) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Container) PreloadFields() *[]string {
	return &containerValidPreloads
}

type ContainerList struct {
	Items []*Container
	BaseList
}

func (objList *ContainerList) GetKind() *string {
	return &containerKind
}

func (obj *ContainerList) PrimaryKeyField() *string {
	return &containerPrimaryKey
}

func (objList *ContainerList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *ContainerList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *ContainerList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *ContainerList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *ContainerList) DataPtr() interface{} {
	return &objList.Items
}

var (
	environmentKind          = "environment"
	environmentTable         = "environments"
	environmentPrimaryKey    = "id"
	environmentValidPreloads = []string{"Cluster", "Creator", "Project", "Applications", "VirtualSpace"}
)

func (obj *Environment) TableName() *string {
	return &environmentTable
}

func (obj *Environment) GetKind() *string {
	return &environmentKind
}

func (obj *Environment) PrimaryKeyField() *string {
	return &environmentPrimaryKey
}

func (obj *Environment) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Environment) PreloadFields() *[]string {
	return &environmentValidPreloads
}

type EnvironmentList struct {
	Items []*Environment
	BaseList
}

func (objList *EnvironmentList) GetKind() *string {
	return &environmentKind
}

func (obj *EnvironmentList) PrimaryKeyField() *string {
	return &environmentPrimaryKey
}

func (objList *EnvironmentList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *EnvironmentList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *EnvironmentList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *EnvironmentList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *EnvironmentList) DataPtr() interface{} {
	return &objList.Items
}

var (
	environmentResourceKind          = "environment_resource"
	environmentResourceTable         = "environment_resources"
	environmentResourcePrimaryKey    = "id"
	environmentResourceValidPreloads = []string{}
)

func (obj *EnvironmentResource) TableName() *string {
	return &environmentResourceTable
}

func (obj *EnvironmentResource) GetKind() *string {
	return &environmentResourceKind
}

func (obj *EnvironmentResource) PrimaryKeyField() *string {
	return &environmentResourcePrimaryKey
}

func (obj *EnvironmentResource) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *EnvironmentResource) PreloadFields() *[]string {
	return &environmentResourceValidPreloads
}

type EnvironmentResourceList struct {
	Items []*EnvironmentResource
	BaseList
}

func (objList *EnvironmentResourceList) GetKind() *string {
	return &environmentResourceKind
}

func (obj *EnvironmentResourceList) PrimaryKeyField() *string {
	return &environmentResourcePrimaryKey
}

func (objList *EnvironmentResourceList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *EnvironmentResourceList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *EnvironmentResourceList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *EnvironmentResourceList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *EnvironmentResourceList) DataPtr() interface{} {
	return &objList.Items
}

var (
	environmentUserRelKind          = "environment_user_rel"
	environmentUserRelTable         = "environment_user_rels"
	environmentUserRelPrimaryKey    = "id"
	environmentUserRelValidPreloads = []string{"User", "Environment"}
)

func (obj *EnvironmentUserRel) TableName() *string {
	return &environmentUserRelTable
}

func (obj *EnvironmentUserRel) GetKind() *string {
	return &environmentUserRelKind
}

func (obj *EnvironmentUserRel) PrimaryKeyField() *string {
	return &environmentUserRelPrimaryKey
}

func (obj *EnvironmentUserRel) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *EnvironmentUserRel) PreloadFields() *[]string {
	return &environmentUserRelValidPreloads
}

type EnvironmentUserRelList struct {
	Items []*EnvironmentUserRel
	BaseList
}

func (objList *EnvironmentUserRelList) GetKind() *string {
	return &environmentUserRelKind
}

func (obj *EnvironmentUserRelList) PrimaryKeyField() *string {
	return &environmentUserRelPrimaryKey
}

func (objList *EnvironmentUserRelList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *EnvironmentUserRelList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *EnvironmentUserRelList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *EnvironmentUserRelList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *EnvironmentUserRelList) DataPtr() interface{} {
	return &objList.Items
}

var (
	logQueryHistoryKind          = "log_query_history"
	logQueryHistoryTable         = "log_query_historys"
	logQueryHistoryPrimaryKey    = "id"
	logQueryHistoryValidPreloads = []string{"Cluster", "Creator"}
)

func (obj *LogQueryHistory) TableName() *string {
	return &logQueryHistoryTable
}

func (obj *LogQueryHistory) GetKind() *string {
	return &logQueryHistoryKind
}

func (obj *LogQueryHistory) PrimaryKeyField() *string {
	return &logQueryHistoryPrimaryKey
}

func (obj *LogQueryHistory) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *LogQueryHistory) PreloadFields() *[]string {
	return &logQueryHistoryValidPreloads
}

type LogQueryHistoryList struct {
	Items []*LogQueryHistory
	BaseList
}

func (objList *LogQueryHistoryList) GetKind() *string {
	return &logQueryHistoryKind
}

func (obj *LogQueryHistoryList) PrimaryKeyField() *string {
	return &logQueryHistoryPrimaryKey
}

func (objList *LogQueryHistoryList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *LogQueryHistoryList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *LogQueryHistoryList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *LogQueryHistoryList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *LogQueryHistoryList) DataPtr() interface{} {
	return &objList.Items
}

var (
	logQuerySnapshotKind          = "log_query_snapshot"
	logQuerySnapshotTable         = "log_query_snapshots"
	logQuerySnapshotPrimaryKey    = "id"
	logQuerySnapshotValidPreloads = []string{"Cluster", "Creator"}
)

func (obj *LogQuerySnapshot) TableName() *string {
	return &logQuerySnapshotTable
}

func (obj *LogQuerySnapshot) GetKind() *string {
	return &logQuerySnapshotKind
}

func (obj *LogQuerySnapshot) PrimaryKeyField() *string {
	return &logQuerySnapshotPrimaryKey
}

func (obj *LogQuerySnapshot) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *LogQuerySnapshot) PreloadFields() *[]string {
	return &logQuerySnapshotValidPreloads
}

type LogQuerySnapshotList struct {
	Items []*LogQuerySnapshot
	BaseList
}

func (objList *LogQuerySnapshotList) GetKind() *string {
	return &logQuerySnapshotKind
}

func (obj *LogQuerySnapshotList) PrimaryKeyField() *string {
	return &logQuerySnapshotPrimaryKey
}

func (objList *LogQuerySnapshotList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *LogQuerySnapshotList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *LogQuerySnapshotList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *LogQuerySnapshotList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *LogQuerySnapshotList) DataPtr() interface{} {
	return &objList.Items
}

var (
	messageKind          = "message"
	messageTable         = "messages"
	messagePrimaryKey    = "id"
	messageValidPreloads = []string{}
)

func (obj *Message) TableName() *string {
	return &messageTable
}

func (obj *Message) GetKind() *string {
	return &messageKind
}

func (obj *Message) PrimaryKeyField() *string {
	return &messagePrimaryKey
}

func (obj *Message) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Message) PreloadFields() *[]string {
	return &messageValidPreloads
}

type MessageList struct {
	Items []*Message
	BaseList
}

func (objList *MessageList) GetKind() *string {
	return &messageKind
}

func (obj *MessageList) PrimaryKeyField() *string {
	return &messagePrimaryKey
}

func (objList *MessageList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *MessageList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *MessageList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *MessageList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *MessageList) DataPtr() interface{} {
	return &objList.Items
}

var (
	openAPPKind          = "open_app"
	openAPPTable         = "open_apps"
	openAPPPrimaryKey    = "id"
	openAPPValidPreloads = []string{}
)

func (obj *OpenAPP) TableName() *string {
	return &openAPPTable
}

func (obj *OpenAPP) GetKind() *string {
	return &openAPPKind
}

func (obj *OpenAPP) PrimaryKeyField() *string {
	return &openAPPPrimaryKey
}

func (obj *OpenAPP) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *OpenAPP) PreloadFields() *[]string {
	return &openAPPValidPreloads
}

type OpenAPPList struct {
	Items []*OpenAPP
	BaseList
}

func (objList *OpenAPPList) GetKind() *string {
	return &openAPPKind
}

func (obj *OpenAPPList) PrimaryKeyField() *string {
	return &openAPPPrimaryKey
}

func (objList *OpenAPPList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *OpenAPPList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *OpenAPPList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *OpenAPPList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *OpenAPPList) DataPtr() interface{} {
	return &objList.Items
}

var (
	projectKind          = "project"
	projectTable         = "projects"
	projectPrimaryKey    = "id"
	projectValidPreloads = []string{"Tenant", "Createor", "Tenant", "Environments"}
)

func (obj *Project) TableName() *string {
	return &projectTable
}

func (obj *Project) GetKind() *string {
	return &projectKind
}

func (obj *Project) PrimaryKeyField() *string {
	return &projectPrimaryKey
}

func (obj *Project) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Project) PreloadFields() *[]string {
	return &projectValidPreloads
}

type ProjectList struct {
	Items []*Project
	BaseList
}

func (objList *ProjectList) GetKind() *string {
	return &projectKind
}

func (obj *ProjectList) PrimaryKeyField() *string {
	return &projectPrimaryKey
}

func (objList *ProjectList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *ProjectList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *ProjectList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *ProjectList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *ProjectList) DataPtr() interface{} {
	return &objList.Items
}

var (
	projectUserRelKind          = "project_user_rel"
	projectUserRelTable         = "project_user_rels"
	projectUserRelPrimaryKey    = "id"
	projectUserRelValidPreloads = []string{"User", "Project"}
)

func (obj *ProjectUserRel) TableName() *string {
	return &projectUserRelTable
}

func (obj *ProjectUserRel) GetKind() *string {
	return &projectUserRelKind
}

func (obj *ProjectUserRel) PrimaryKeyField() *string {
	return &projectUserRelPrimaryKey
}

func (obj *ProjectUserRel) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *ProjectUserRel) PreloadFields() *[]string {
	return &projectUserRelValidPreloads
}

type ProjectUserRelList struct {
	Items []*ProjectUserRel
	BaseList
}

func (objList *ProjectUserRelList) GetKind() *string {
	return &projectUserRelKind
}

func (obj *ProjectUserRelList) PrimaryKeyField() *string {
	return &projectUserRelPrimaryKey
}

func (objList *ProjectUserRelList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *ProjectUserRelList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *ProjectUserRelList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *ProjectUserRelList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *ProjectUserRelList) DataPtr() interface{} {
	return &objList.Items
}

var (
	registryKind          = "registry"
	registryTable         = "registrys"
	registryPrimaryKey    = "id"
	registryValidPreloads = []string{"Project"}
)

func (obj *Registry) TableName() *string {
	return &registryTable
}

func (obj *Registry) GetKind() *string {
	return &registryKind
}

func (obj *Registry) PrimaryKeyField() *string {
	return &registryPrimaryKey
}

func (obj *Registry) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Registry) PreloadFields() *[]string {
	return &registryValidPreloads
}

type RegistryList struct {
	Items []*Registry
	BaseList
}

func (objList *RegistryList) GetKind() *string {
	return &registryKind
}

func (obj *RegistryList) PrimaryKeyField() *string {
	return &registryPrimaryKey
}

func (objList *RegistryList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *RegistryList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *RegistryList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *RegistryList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *RegistryList) DataPtr() interface{} {
	return &objList.Items
}

var (
	systemRoleKind          = "system_role"
	systemRoleTable         = "system_roles"
	systemRolePrimaryKey    = "id"
	systemRoleValidPreloads = []string{"Users"}
)

func (obj *SystemRole) TableName() *string {
	return &systemRoleTable
}

func (obj *SystemRole) GetKind() *string {
	return &systemRoleKind
}

func (obj *SystemRole) PrimaryKeyField() *string {
	return &systemRolePrimaryKey
}

func (obj *SystemRole) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *SystemRole) PreloadFields() *[]string {
	return &systemRoleValidPreloads
}

type SystemRoleList struct {
	Items []*SystemRole
	BaseList
}

func (objList *SystemRoleList) GetKind() *string {
	return &systemRoleKind
}

func (obj *SystemRoleList) PrimaryKeyField() *string {
	return &systemRolePrimaryKey
}

func (objList *SystemRoleList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *SystemRoleList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *SystemRoleList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *SystemRoleList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *SystemRoleList) DataPtr() interface{} {
	return &objList.Items
}

var (
	tenantKind          = "tenant"
	tenantTable         = "tenants"
	tenantPrimaryKey    = "id"
	tenantValidPreloads = []string{"Users", "Projects"}
)

func (obj *Tenant) TableName() *string {
	return &tenantTable
}

func (obj *Tenant) GetKind() *string {
	return &tenantKind
}

func (obj *Tenant) PrimaryKeyField() *string {
	return &tenantPrimaryKey
}

func (obj *Tenant) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Tenant) PreloadFields() *[]string {
	return &tenantValidPreloads
}

type TenantList struct {
	Items []*Tenant
	BaseList
}

func (objList *TenantList) GetKind() *string {
	return &tenantKind
}

func (obj *TenantList) PrimaryKeyField() *string {
	return &tenantPrimaryKey
}

func (objList *TenantList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *TenantList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *TenantList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *TenantList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *TenantList) DataPtr() interface{} {
	return &objList.Items
}

var (
	tenantResourceQuotaKind          = "tenant_resource_quota"
	tenantResourceQuotaTable         = "tenant_resource_quotas"
	tenantResourceQuotaPrimaryKey    = "id"
	tenantResourceQuotaValidPreloads = []string{"Tenant", "Cluster", "TenantResourceQuotaApply"}
)

func (obj *TenantResourceQuota) TableName() *string {
	return &tenantResourceQuotaTable
}

func (obj *TenantResourceQuota) GetKind() *string {
	return &tenantResourceQuotaKind
}

func (obj *TenantResourceQuota) PrimaryKeyField() *string {
	return &tenantResourceQuotaPrimaryKey
}

func (obj *TenantResourceQuota) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *TenantResourceQuota) PreloadFields() *[]string {
	return &tenantResourceQuotaValidPreloads
}

type TenantResourceQuotaList struct {
	Items []*TenantResourceQuota
	BaseList
}

func (objList *TenantResourceQuotaList) GetKind() *string {
	return &tenantResourceQuotaKind
}

func (obj *TenantResourceQuotaList) PrimaryKeyField() *string {
	return &tenantResourceQuotaPrimaryKey
}

func (objList *TenantResourceQuotaList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *TenantResourceQuotaList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *TenantResourceQuotaList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *TenantResourceQuotaList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *TenantResourceQuotaList) DataPtr() interface{} {
	return &objList.Items
}

var (
	tenantResourceQuotaApplyKind          = "tenant_resource_quota_apply"
	tenantResourceQuotaApplyTable         = "tenant_resource_quota_applys"
	tenantResourceQuotaApplyPrimaryKey    = "id"
	tenantResourceQuotaApplyValidPreloads = []string{}
)

func (obj *TenantResourceQuotaApply) TableName() *string {
	return &tenantResourceQuotaApplyTable
}

func (obj *TenantResourceQuotaApply) GetKind() *string {
	return &tenantResourceQuotaApplyKind
}

func (obj *TenantResourceQuotaApply) PrimaryKeyField() *string {
	return &tenantResourceQuotaApplyPrimaryKey
}

func (obj *TenantResourceQuotaApply) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *TenantResourceQuotaApply) PreloadFields() *[]string {
	return &tenantResourceQuotaApplyValidPreloads
}

type TenantResourceQuotaApplyList struct {
	Items []*TenantResourceQuotaApply
	BaseList
}

func (objList *TenantResourceQuotaApplyList) GetKind() *string {
	return &tenantResourceQuotaApplyKind
}

func (obj *TenantResourceQuotaApplyList) PrimaryKeyField() *string {
	return &tenantResourceQuotaApplyPrimaryKey
}

func (objList *TenantResourceQuotaApplyList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *TenantResourceQuotaApplyList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *TenantResourceQuotaApplyList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *TenantResourceQuotaApplyList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *TenantResourceQuotaApplyList) DataPtr() interface{} {
	return &objList.Items
}

var (
	tenantUserRelKind          = "tenant_user_rel"
	tenantUserRelTable         = "tenant_user_rels"
	tenantUserRelPrimaryKey    = "id"
	tenantUserRelValidPreloads = []string{"User", "Tenant"}
)

func (obj *TenantUserRel) TableName() *string {
	return &tenantUserRelTable
}

func (obj *TenantUserRel) GetKind() *string {
	return &tenantUserRelKind
}

func (obj *TenantUserRel) PrimaryKeyField() *string {
	return &tenantUserRelPrimaryKey
}

func (obj *TenantUserRel) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *TenantUserRel) PreloadFields() *[]string {
	return &tenantUserRelValidPreloads
}

type TenantUserRelList struct {
	Items []*TenantUserRel
	BaseList
}

func (objList *TenantUserRelList) GetKind() *string {
	return &tenantUserRelKind
}

func (obj *TenantUserRelList) PrimaryKeyField() *string {
	return &tenantUserRelPrimaryKey
}

func (objList *TenantUserRelList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *TenantUserRelList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *TenantUserRelList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *TenantUserRelList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *TenantUserRelList) DataPtr() interface{} {
	return &objList.Items
}

var (
	userKind          = "user"
	userTable         = "users"
	userPrimaryKey    = "id"
	userValidPreloads = []string{"SystemRole"}
)

func (obj *User) TableName() *string {
	return &userTable
}

func (obj *User) GetKind() *string {
	return &userKind
}

func (obj *User) PrimaryKeyField() *string {
	return &userPrimaryKey
}

func (obj *User) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *User) PreloadFields() *[]string {
	return &userValidPreloads
}

type UserList struct {
	Items []*User
	BaseList
}

func (objList *UserList) GetKind() *string {
	return &userKind
}

func (obj *UserList) PrimaryKeyField() *string {
	return &userPrimaryKey
}

func (objList *UserList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *UserList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *UserList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *UserList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *UserList) DataPtr() interface{} {
	return &objList.Items
}

var (
	userMessageStatusKind          = "user_message_status"
	userMessageStatusTable         = "user_message_status"
	userMessageStatusPrimaryKey    = "id"
	userMessageStatusValidPreloads = []string{}
)

func (obj *UserMessageStatus) TableName() *string {
	return &userMessageStatusTable
}

func (obj *UserMessageStatus) GetKind() *string {
	return &userMessageStatusKind
}

func (obj *UserMessageStatus) PrimaryKeyField() *string {
	return &userMessageStatusPrimaryKey
}

func (obj *UserMessageStatus) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *UserMessageStatus) PreloadFields() *[]string {
	return &userMessageStatusValidPreloads
}

type UserMessageStatusList struct {
	Items []*UserMessageStatus
	BaseList
}

func (objList *UserMessageStatusList) GetKind() *string {
	return &userMessageStatusKind
}

func (obj *UserMessageStatusList) PrimaryKeyField() *string {
	return &userMessageStatusPrimaryKey
}

func (objList *UserMessageStatusList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *UserMessageStatusList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *UserMessageStatusList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *UserMessageStatusList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *UserMessageStatusList) DataPtr() interface{} {
	return &objList.Items
}

var (
	virtualDomainKind          = "virtual_domain"
	virtualDomainTable         = "virtual_domains"
	virtualDomainPrimaryKey    = "id"
	virtualDomainValidPreloads = []string{}
)

func (obj *VirtualDomain) TableName() *string {
	return &virtualDomainTable
}

func (obj *VirtualDomain) GetKind() *string {
	return &virtualDomainKind
}

func (obj *VirtualDomain) PrimaryKeyField() *string {
	return &virtualDomainPrimaryKey
}

func (obj *VirtualDomain) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *VirtualDomain) PreloadFields() *[]string {
	return &virtualDomainValidPreloads
}

type VirtualDomainList struct {
	Items []*VirtualDomain
	BaseList
}

func (objList *VirtualDomainList) GetKind() *string {
	return &virtualDomainKind
}

func (obj *VirtualDomainList) PrimaryKeyField() *string {
	return &virtualDomainPrimaryKey
}

func (objList *VirtualDomainList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *VirtualDomainList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *VirtualDomainList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *VirtualDomainList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *VirtualDomainList) DataPtr() interface{} {
	return &objList.Items
}

var (
	virtualSpaceKind          = "virtual_space"
	virtualSpaceTable         = "virtual_spaces"
	virtualSpacePrimaryKey    = "id"
	virtualSpaceValidPreloads = []string{}
)

func (obj *VirtualSpace) TableName() *string {
	return &virtualSpaceTable
}

func (obj *VirtualSpace) GetKind() *string {
	return &virtualSpaceKind
}

func (obj *VirtualSpace) PrimaryKeyField() *string {
	return &virtualSpacePrimaryKey
}

func (obj *VirtualSpace) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *VirtualSpace) PreloadFields() *[]string {
	return &virtualSpaceValidPreloads
}

type VirtualSpaceList struct {
	Items []*VirtualSpace
	BaseList
}

func (objList *VirtualSpaceList) GetKind() *string {
	return &virtualSpaceKind
}

func (obj *VirtualSpaceList) PrimaryKeyField() *string {
	return &virtualSpacePrimaryKey
}

func (objList *VirtualSpaceList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *VirtualSpaceList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *VirtualSpaceList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *VirtualSpaceList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *VirtualSpaceList) DataPtr() interface{} {
	return &objList.Items
}

var (
	virtualSpaceUserRelKind          = "virtual_space_user_rel"
	virtualSpaceUserRelTable         = "virtual_space_user_rels"
	virtualSpaceUserRelPrimaryKey    = "id"
	virtualSpaceUserRelValidPreloads = []string{}
)

func (obj *VirtualSpaceUserRel) TableName() *string {
	return &virtualSpaceUserRelTable
}

func (obj *VirtualSpaceUserRel) GetKind() *string {
	return &virtualSpaceUserRelKind
}

func (obj *VirtualSpaceUserRel) PrimaryKeyField() *string {
	return &virtualSpaceUserRelPrimaryKey
}

func (obj *VirtualSpaceUserRel) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *VirtualSpaceUserRel) PreloadFields() *[]string {
	return &virtualSpaceUserRelValidPreloads
}

type VirtualSpaceUserRelList struct {
	Items []*VirtualSpaceUserRel
	BaseList
}

func (objList *VirtualSpaceUserRelList) GetKind() *string {
	return &virtualSpaceUserRelKind
}

func (obj *VirtualSpaceUserRelList) PrimaryKeyField() *string {
	return &virtualSpaceUserRelPrimaryKey
}

func (objList *VirtualSpaceUserRelList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *VirtualSpaceUserRelList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *VirtualSpaceUserRelList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *VirtualSpaceUserRelList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *VirtualSpaceUserRelList) DataPtr() interface{} {
	return &objList.Items
}

var (
	workloadKind          = "workload"
	workloadTable         = "workloads"
	workloadPrimaryKey    = "id"
	workloadValidPreloads = []string{"Containers"}
)

func (obj *Workload) TableName() *string {
	return &workloadTable
}

func (obj *Workload) GetKind() *string {
	return &workloadKind
}

func (obj *Workload) PrimaryKeyField() *string {
	return &workloadPrimaryKey
}

func (obj *Workload) PrimaryKeyValue() interface{} {
	return obj.ID
}

func (obj *Workload) PreloadFields() *[]string {
	return &workloadValidPreloads
}

type WorkloadList struct {
	Items []*Workload
	BaseList
}

func (objList *WorkloadList) GetKind() *string {
	return &workloadKind
}

func (obj *WorkloadList) PrimaryKeyField() *string {
	return &workloadPrimaryKey
}

func (objList *WorkloadList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *WorkloadList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *WorkloadList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *WorkloadList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *WorkloadList) DataPtr() interface{} {
	return &objList.Items
}
