package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redisv8 "github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/redis"
)

var redisinstance *redis.Client

// models.User 的 hook 上用到了 redis 实例，需要在使用前初始化这个redis 实例
func InitRedis(cli *redis.Client) error {
	redisinstance = cli
	return nil
}

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

func (c *CacheLayer) GetUserAuthority(user *User) *UserAuthority {
	authinfo := UserAuthority{}
	if err := c.Redis.Client.Get(context.Background(), UserAuthorityKey(user.Username)).Scan(&authinfo); err != nil {
		log.Debugf("authorization data for user %v not exist, refresh now, if will timeout in %v minutes", user.Username, userAuthorizationDataExpireMinute)
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

func (c *CacheLayer) FlushUserAuthority(user *User) *UserAuthority {
	auth := new(UserAuthority)

	sysrole := SystemRole{ID: user.SystemRoleID}
	if err := c.DataBase.DB().First(&sysrole).Error; err != nil {
		log.Errorf("flush user authority err: %v", err)
	}

	var turs []TenantUserRels
	if err := c.DataBase.DB().Preload("Tenant").Where("user_id = ?", user.ID).Find(&turs).Error; err != nil {
		log.Errorf("query db: %v", err)
	}
	var purs []ProjectUserRels
	if err := c.DataBase.DB().Preload("Project").Where("user_id = ?", user.ID).Find(&purs).Error; err != nil {
		log.Errorf("query db: %v", err)
	}
	var eurs []EnvironmentUserRels
	if err := c.DataBase.DB().Preload("Environment").Where("user_id = ?", user.ID).Find(&eurs).Error; err != nil {
		log.Errorf("query db: %v", err)
	}
	var vurs []VirtualSpaceUserRels
	if err := c.DataBase.DB().Preload("VirtualSpace").Where("user_id = ?", user.ID).Find(&vurs).Error; err != nil {
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

	c.Redis.Client.SetEX(context.Background(), UserAuthorityKey(user.Username), auth, time.Duration(userAuthorizationDataExpireMinute)*time.Minute)
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

const (
	GlobalResourceTreeKey = "globalResoruceCacheTree"
	GlobalResourceLock    = "globalResoruceCacheTreeLock"
)

var GlobalResourceDuration = time.Second * 3

type GlobalResourceTree struct {
	Tree   *ResourceNode
	Locker *Locker
	rdb    *redis.Client
	mdb    *gorm.DB
}

func GetGlobalResourceTree(rdb *redis.Client, db *gorm.DB) *GlobalResourceTree {
	if _globalResourceTree != nil {
		return _globalResourceTree
	}
	t := &GlobalResourceTree{
		rdb: rdb,
		mdb: db,
		Locker: &Locker{
			Name:    GlobalResourceLock,
			Timeout: 1 * time.Minute,
			rdb:     rdb,
		},
		Tree: &ResourceNode{},
	}
	if t.Refresh() {
		t.Locker.Lock(context.Background())
		t.SetCache()
		t.Locker.UnLock(context.Background())
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
	t.Locker.Lock(context.Background())
	defer t.Locker.UnLock(context.Background())
	t.Refresh()
	if t.Tree.AddOrUpdateChild(ResTenant, tid, name) {
		t.SetCache()
	}
}

func (t *GlobalResourceTree) DelTenant(tid uint) {
	t.Locker.Lock(context.Background())
	defer t.Locker.UnLock(context.Background())
	t.Refresh()
	if t.Tree.DelChild(ResTenant, tid) {
		t.SetCache()
	}
}

func (t *GlobalResourceTree) UpsertVirtualSpace(vid uint, name string) {
	t.Locker.Lock(context.Background())
	defer t.Locker.UnLock(context.Background())
	t.Refresh()
	if t.Tree.AddOrUpdateChild(ResVirtualSpace, vid, name) {
		t.SetCache()
	}
}

func (t *GlobalResourceTree) DelVirtualSpace(vid uint) {
	t.Locker.Lock(context.Background())
	defer t.Locker.UnLock(context.Background())
	t.Refresh()
	if t.Tree.DelChild(ResVirtualSpace, vid) {
		t.SetCache()
	}
}

func (t *GlobalResourceTree) UpsertProject(tid, pid uint, name string) error {
	t.Locker.Lock(context.Background())
	defer t.Locker.UnLock(context.Background())
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
	t.Locker.Lock(context.Background())
	defer t.Locker.UnLock(context.Background())
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
	t.Locker.Lock(context.Background())
	defer t.Locker.UnLock(context.Background())
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
	t.Locker.Lock(context.Background())
	defer t.Locker.UnLock(context.Background())
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

// 带阻塞的redis分布式锁 reference: https://www.zhihu.com/question/440583752/answer/1953369763
type Locker struct {
	Name    string
	Timeout time.Duration
	rdb     *redis.Client
}

func (l *Locker) Lock(ctx context.Context) (v string, err error) {
	v, err = l.rdb.Get(ctx, "lock_"+l.Name).Result()
	if err != nil && err != redisv8.Nil {
		// 报错
		return
	} else if err == redisv8.Nil {
		//  不存在KEY，即正常获取到了锁
		err = l.rdb.Set(ctx, "lock_"+l.Name, time.Now(), l.Timeout).Err()
		log.Infof("locking redis mutex: %s", l.Name)
	} else {
		// 阻塞， 等待锁释放的通知
		log.Infof("wait redis mutex: %s", l.Name)
		_, err = l.rdb.BRPop(ctx, l.Timeout, "free_"+l.Name).Result()
		l.rdb.Del(ctx, "free_"+l.Name)
	}
	return
}

func (l *Locker) UnLock(ctx context.Context) (err error) {
	_, err = l.rdb.Get(ctx, "lock_"+l.Name).Result()
	if err == redisv8.Nil {
		//  不存在KEY，即锁不存在
		return redisv8.Nil
	}
	// 存在锁, push一个值，让其他线程将有一个能得到锁
	log.Infof("unlock redis mutex: %s", l.Name)
	_, err = l.rdb.LPush(ctx, "free_"+l.Name, time.Now()).Result()
	if err != nil {
		return
	}
	// 自己释放锁
	l.rdb.Del(ctx, "lock_"+l.Name)
	return
}
