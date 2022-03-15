package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/redis"
)

type CacheLayer struct {
	DataBase *database.Database
	Redis    *redis.Client
}

var userAuthorizationDataExpireMinute int64 = 60

func (c *CacheLayer) RefreshUserAuthority(u *User) {}

func UserAuthorityKey(username string) string {
	return fmt.Sprintf("authorization_data__%s", username)
}

type UserResource struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	IsAdmin bool   `json:"isAdmin"`
}

type UserAuthority struct {
	IsSystemAdmin bool            `json:"isSystemAdmin"`
	Tenants       []*UserResource `json:"tenant"`
	Projects      []*UserResource `json:"projects"`
	Environments  []*UserResource `json:"environments"`
	VirtualSpaces []*UserResource `json:"virtualSpaces"`
}

func (c *CacheLayer) GetUserAuthority(user CommonUserIface) *UserAuthority {
	authinfo := UserAuthority{}
	if err := c.Redis.Client.Get(context.Background(), UserAuthorityKey(user.GetUsername())).Scan(&authinfo); err != nil {
		log.Debugf("authorization data for user %v not exist, refresh now, if will timeout in %v minutes", user.GetUsername(), userAuthorizationDataExpireMinute)
		newAuthInfo := c.FlushUserAuthority(user)
		return newAuthInfo
	}
	return &authinfo
}

func (c *CacheLayer) GetGlobalResourceTree() *GlobalResourceTree {
	return GetGlobalResourceTree(c.Redis, c.DataBase.DB())
}

func (auth *UserAuthority) MarshalBinary() ([]byte, error) {
	return json.Marshal(auth)
}

func (auth *UserAuthority) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &auth)
}

func (c *CacheLayer) FlushUserAuthority(user CommonUserIface) *UserAuthority {
	auth := new(UserAuthority)

	sysrole := SystemRole{ID: user.GetSystemRoleID()}
	if err := c.DataBase.DB().First(&sysrole).Error; err != nil {
		log.Errorf("flush user authority err: %v", err)
	}

	var turs []TenantUserRels
	if err := c.DataBase.DB().Preload("Tenant").Where("user_id = ?", user.GetID()).Find(&turs).Error; err != nil {
		log.Errorf("query db: %v", err)
	}
	var purs []ProjectUserRels
	if err := c.DataBase.DB().Preload("Project").Where("user_id = ?", user.GetID()).Find(&purs).Error; err != nil {
		log.Errorf("query db: %v", err)
	}
	var eurs []EnvironmentUserRels
	if err := c.DataBase.DB().Preload("Environment").Where("user_id = ?", user.GetID()).Find(&eurs).Error; err != nil {
		log.Errorf("query db: %v", err)
	}
	var vurs []VirtualSpaceUserRels
	if err := c.DataBase.DB().Preload("VirtualSpace").Where("user_id = ?", user.GetID()).Find(&vurs).Error; err != nil {
		log.Errorf("query db: %v", err)
	}

	auth.IsSystemAdmin = sysrole.RoleCode == SystemRoleAdmin
	auth.Tenants = make([]*UserResource, len(turs))
	auth.Projects = make([]*UserResource, len(purs))
	auth.Environments = make([]*UserResource, len(eurs))
	auth.VirtualSpaces = make([]*UserResource, len(vurs))

	for i := range turs {
		tmp := turs[i]
		auth.Tenants[i] = &UserResource{
			ID:      int(tmp.TenantID),
			Name:    tmp.Tenant.TenantName,
			Role:    tmp.Role,
			IsAdmin: tmp.Role == TenantRoleAdmin,
		}
	}
	for i := range purs {
		tmp := purs[i]
		auth.Projects[i] = &UserResource{
			ID:      int(tmp.ProjectID),
			Name:    tmp.Project.ProjectName,
			Role:    tmp.Role,
			IsAdmin: tmp.Role == ProjectRoleAdmin,
		}
	}
	for i := range eurs {
		tmp := eurs[i]
		auth.Environments[i] = &UserResource{
			ID:      int(tmp.EnvironmentID),
			Name:    tmp.Environment.EnvironmentName,
			Role:    tmp.Role,
			IsAdmin: tmp.Role == EnvironmentRoleOperator,
		}
	}
	for i := range vurs {
		tmp := vurs[i]
		auth.VirtualSpaces[i] = &UserResource{
			ID:      int(tmp.VirtualSpaceID),
			Name:    tmp.VirtualSpace.VirtualSpaceName,
			Role:    tmp.Role,
			IsAdmin: tmp.Role == VirtualSpaceRoleAdmin,
		}
	}

	c.Redis.Client.SetEX(context.Background(), UserAuthorityKey(user.GetUsername()), auth, time.Duration(userAuthorizationDataExpireMinute)*time.Minute)
	return auth
}

func (auth *UserAuthority) GetResourceRole(kind string, id uint) string {
	switch kind {
	case ResTenant:
		for _, tenant := range auth.Tenants {
			if id == uint(tenant.ID) {
				return tenant.Role
			}
		}
	case ResProject:
		for _, proj := range auth.Projects {
			if id == uint(proj.ID) {
				return proj.Role
			}
		}
	case ResEnvironment:
		for _, env := range auth.Environments {
			if id == uint(env.ID) {
				return env.Role
			}
		}
	case ResVirtualSpace:
		for _, vs := range auth.VirtualSpaces {
			if id == uint(vs.ID) {
				return vs.Role
			}
		}
	}
	return ""
}

func (auth *UserAuthority) IsTenantAdmin(tenantid uint) bool {
	role := auth.GetResourceRole(ResTenant, tenantid)
	return role == TenantRoleAdmin
}

func (auth *UserAuthority) IsProjectAdmin(projectid uint) bool {
	role := auth.GetResourceRole(ResProject, projectid)
	return role == ProjectRoleAdmin
}

func (auth *UserAuthority) IsProjectOps(projectid uint) bool {
	role := auth.GetResourceRole(ResProject, projectid)
	return role == ProjectRoleOps
}

func (auth *UserAuthority) IsEnvironmentOperator(envid uint) bool {
	role := auth.GetResourceRole(ResEnvironment, envid)
	return role == EnvironmentRoleOperator
}

func (auth *UserAuthority) IsVirtualSpaceAdmin(vsid uint) bool {
	role := auth.GetResourceRole(ResVirtualSpace, vsid)
	return role == VirtualSpaceRoleAdmin
}

// 全局 租户-项目-环境 缓存

var _globalResourceTree *GlobalResourceTree

const GlobalResourceTreeKey = "globalResoruceCacheTree"

type GlobalResourceTree struct {
	Tree *ResourceNode
	rdb  *redis.Client
	mdb  *gorm.DB
}

func GetGlobalResourceTree(rdb *redis.Client, db *gorm.DB) *GlobalResourceTree {
	if _globalResourceTree != nil {
		return _globalResourceTree
	}
	t := &GlobalResourceTree{
		rdb:  rdb,
		mdb:  db,
		Tree: &ResourceNode{},
	}
	if t.Refresh() {
		t.SetCache()
	}
	return t
}

func (n *ResourceNode) MarshalBinary() ([]byte, error) {
	return json.Marshal(*n)
}

func (n *ResourceNode) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, n)
}

func (t *GlobalResourceTree) BuildRootResourceNodeFromDB() {
	var (
		tenants      []Tenant
		projects     []Project
		environments []Environment
	)
	t.mdb.Find(&tenants)
	t.mdb.Find(&projects)
	t.mdb.Preload("Cluster").Find(&environments)
	for _, tenant := range tenants {
		t.Tree.AddOrUpdateChild(ResTenant, tenant.ID, tenant.TenantName)
	}
	for _, project := range projects {
		tNode := t.Tree.FindNode(ResTenant, project.TenantID)
		if tNode != nil {
			tNode.AddOrUpdateChild(ResProject, project.ID, project.ProjectName)
		}
	}
	for _, env := range environments {
		pNode := t.Tree.FindNode(ResProject, env.ProjectID)
		if pNode != nil {
			pNode.AddOrUpdateChild(ResEnvironment, env.ID, env.EnvironmentName, env.Cluster.ClusterName, env.Namespace)
		}
	}
}

func (t *GlobalResourceTree) Refresh() bool {
	if err := t.rdb.Get(context.Background(), GlobalResourceTreeKey).Scan(t.Tree); err != nil {
		log.Warnf("failed to get global resource tree from redis %v ; will load from database", err)
		t.BuildRootResourceNodeFromDB()
		return true
	}
	return false
}

func (t *GlobalResourceTree) SetCache() {
	if _, err := t.rdb.Set(context.Background(), GlobalResourceTreeKey, t.Tree, 0).Result(); err != nil {
		log.Warnf("failed to set global resource tree  %v;", err)
	}
}

func (t *GlobalResourceTree) UpsertTenant(tid uint, name string) {
	t.Refresh()
	if t.Tree.AddOrUpdateChild(ResTenant, tid, name) {
		t.SetCache()
	}
}

func (t *GlobalResourceTree) DelTenant(tid uint) {
	t.Refresh()
	if t.Tree.DelChild(ResTenant, tid) {
		t.SetCache()
	}
}

func (t *GlobalResourceTree) UpsertVirtualSpace(vid uint, name string) {
	t.Refresh()
	if t.Tree.AddOrUpdateChild(ResVirtualSpace, vid, name) {
		t.SetCache()
	}
}

func (t *GlobalResourceTree) DelVirtualSpace(vid uint) {
	t.Refresh()
	if t.Tree.DelChild(ResVirtualSpace, vid) {
		t.SetCache()
	}
}

func (t *GlobalResourceTree) UpsertProject(tid, pid uint, name string) error {
	t.Refresh()
	tnode := t.Tree.FindNode(ResTenant, tid)
	if tnode == nil {
		return fmt.Errorf("添加项目节点失败，上游租户 %v 不存在", tid)
	}
	if tnode.AddOrUpdateChild(ResProject, pid, name) {
		t.SetCache()
	}
	return nil
}

func (t *GlobalResourceTree) DelProject(tid, pid uint) error {
	t.Refresh()
	tnode := t.Tree.FindNode(ResTenant, tid)
	if tnode == nil {
		return fmt.Errorf("删除项目节点失败，上游租户 %v 不存在", tid)
	}
	if tnode.DelChild(ResProject, pid) {
		t.SetCache()
	}
	return nil
}

func (t *GlobalResourceTree) UpsertEnvironment(pid, eid uint, name, cluster, namespace string) error {
	t.Refresh()
	pnode := t.Tree.FindNode(ResProject, pid)
	if pnode == nil {
		return fmt.Errorf("添加环境节点失败，上游项目%v 不存在", pid)
	}
	if pnode.AddOrUpdateChild(ResEnvironment, eid, name, cluster, namespace) {
		t.SetCache()
	}
	return nil
}

func (t *GlobalResourceTree) DelEnvironment(pid, eid uint) error {
	t.Refresh()
	pnode := t.Tree.FindNode(ResProject, pid)
	if pnode == nil {
		return fmt.Errorf("删除环境节点失败，上游项目 %v 不存在", pid)
	}
	if pnode.DelChild(ResEnvironment, eid) {
		t.SetCache()
	}
	return nil
}

type ResourceNode struct {
	Kind      string        `json:"kind,omitempty"`
	ID        uint          `json:"id,omitempty"`
	Name      string        `json:"name,omitempty"`
	Cluster   string        `json:"cluster,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
	Children  ResourceQueue `json:"chlidren,omitempty"`
}

type ResourceQueue []*ResourceNode

func (n *ResourceNode) FindParents(kind string, id uint) ResourceQueue {
	return findParent(n, kind, id)
}

func (n *ResourceNode) FindNode(kind string, id uint) *ResourceNode {
	return n.findNode(kind, id)
}

func (n *ResourceNode) FindNodeByClusterNamespace(cluster, namespace string) *ResourceNode {
	return n.findNodeByClusterNamespace(cluster, namespace)
}

func (n *ResourceNode) Equal(kind string, id uint) bool {
	return n.Kind == kind && n.ID == id
}

func (n *ResourceNode) EqualOpts(name, cluster, namespace string) bool {
	return n.Name == name && n.Cluster == cluster && n.Namespace == namespace
}

func (n *ResourceNode) AddOrUpdateChild(kind string, id uint, opts ...string) bool {
	name := ""
	cluster := ""
	namespace := ""
	switch len(opts) {
	case 1:
		name = opts[0]
	case 2:
		name = opts[0]
		cluster = opts[1]
	case 3:
		name = opts[0]
		cluster = opts[1]
		namespace = opts[2]
	}
	for _, node := range n.Children {
		if node.Equal(kind, id) {
			if node.EqualOpts(name, cluster, namespace) {
				return false
			} else {
				node.Name = name
				node.Cluster = cluster
				node.Namespace = namespace
			}
			return true
		}
	}
	n.Children.Push(&ResourceNode{Kind: kind, ID: id, Name: name, Cluster: cluster, Namespace: namespace})
	return true
}

func (n *ResourceNode) DelChild(kind string, id uint) bool {
	for idx, node := range n.Children {
		if node.Equal(kind, id) {
			n.Children = append(n.Children[:idx], n.Children[idx+1:]...)
			return true
		}
	}
	return false
}

func (q *ResourceQueue) Push(n *ResourceNode) {
	*q = append(*q, n)
}

func (q *ResourceQueue) Pop() {
	length := len(*q)
	if length == 0 {
		return
	}
	*q = (*q)[:length-1]
}

func (n *ResourceNode) findNode(kind string, id uint) *ResourceNode {
	// dfs
	for _, n := range n.Children {
		if r := n.findNode(kind, id); r != nil {
			return r
		}
		if n.Equal(kind, id) {
			return n
		}
	}
	return nil
}

func (n *ResourceNode) findNodeByClusterNamespace(cluster, namespace string) *ResourceNode {
	for _, n := range n.Children {
		if n.Cluster == cluster && n.Namespace == namespace {
			return n
		}
		if r := n.findNodeByClusterNamespace(cluster, namespace); r != nil {
			return r
		}
	}
	return nil
}

func findParent(n *ResourceNode, kind string, id uint) ResourceQueue {
	// dfs
	q := ResourceQueue{}
	for _, n := range n.Children {
		q.Push(n)
		if !n.Equal(kind, id) {
			ret := findParent(n, kind, id)
			if len(ret) == 0 {
				q.Pop()
			} else {
				q = append(q, ret...)
				return q
			}
		} else {
			return q
		}
	}
	return q
}
