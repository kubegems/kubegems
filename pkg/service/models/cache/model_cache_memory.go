// Copyright 2024 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"errors"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
)

const InMemoryModelCacheSize = 128

var ErrCacheEntryNotFound = errors.New("cache entry not found")

var _ ModelCache = &InMemoryModelCache{}

func NewMemoryModelCache(db *gorm.DB) *InMemoryModelCache {
	return &InMemoryModelCache{
		DB:      db,
		dataMap: make(map[string]*Entity),
		mu:      sync.RWMutex{},
		authorities: expirable.NewLRU(InMemoryModelCacheSize, func(k string, v *UserAuthority) {
			log.Info("authority expired", "key", k)
		}, time.Duration(userAuthorizationDataExpireMinute)*time.Minute),
	}
}

type InMemoryModelCache struct {
	DB          *gorm.DB
	dataMap     map[string]*Entity
	mu          sync.RWMutex
	authorities *expirable.LRU[string, *UserAuthority]
}

// BuildCacheIfNotExist implements ModelCache.
func (i *InMemoryModelCache) BuildCacheIfNotExist() error {
	tenants := []models.Tenant{}
	if err := i.DB.Find(&tenants).Error; err != nil {
		return err
	}
	dataMap := make(map[string]*Entity)
	for _, tenant := range tenants {
		n := &Entity{Name: tenant.TenantName, Kind: models.ResTenant, ID: tenant.ID}
		dataMap[n.cacheKey()] = n
	}

	projects := []models.Project{}
	if err := i.DB.Find(&projects).Error; err != nil {
		return err
	}
	for _, project := range projects {
		n := &Entity{Name: project.ProjectName, Kind: models.ResProject, ID: project.ID, Owner: []*Entity{{Kind: models.ResTenant, ID: project.TenantID}}}
		dataMap[n.cacheKey()] = n
	}

	envs := []models.Environment{}
	if err := i.DB.Preload("Cluster").Find(&envs).Error; err != nil {
		return err
	}
	for _, env := range envs {
		if env.Cluster == nil {
			continue
		}
		n := &Entity{
			Name:      env.EnvironmentName,
			Kind:      models.ResEnvironment,
			ID:        env.ID,
			Namespace: env.Namespace,
			Cluster:   env.Cluster.ClusterName,
			Owner:     []*Entity{{Kind: models.ResProject, ID: env.ProjectID}},
		}
		dataMap[n.cacheKey()] = n
		dataMap[envCacheKey(n.Cluster, n.Namespace)] = n
	}
	vspaces := []models.VirtualSpace{}
	if err := i.DB.Find(&vspaces).Error; err != nil {
		return err
	}
	for _, vspace := range vspaces {
		n := &Entity{Name: vspace.VirtualSpaceName, Kind: models.ResVirtualSpace, ID: vspace.ID, Owner: []*Entity{}}
		dataMap[n.cacheKey()] = n
	}
	if len(dataMap) == 0 {
		log.Info("empty cache data")
		return nil
	}
	log.Info("cache data", "data", dataMap)
	syncmap := &sync.Map{}
	for k, v := range dataMap {
		syncmap.Store(k, v)
	}
	i.dataMap = dataMap
	return nil
}

// FindEnvironment implements ModelCache.
func (i *InMemoryModelCache) FindEnvironment(cluster string, namespace string) CommonResourceIface {
	return i.get(envCacheKey(cluster, namespace))
}

// UpsertEnvironment implements ModelCache.
func (i *InMemoryModelCache) UpsertEnvironment(pid uint, eid uint, name string, cluster string, namespace string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	n := Entity{
		Name: name, Kind: models.ResEnvironment, ID: eid, Namespace: namespace, Cluster: cluster,
		Owner: []*Entity{{Kind: models.ResProject, ID: pid}},
	}
	i.dataMap[n.cacheKey()] = &n
	i.dataMap[envCacheKey(cluster, namespace)] = &n
	return nil
}

// DelEnvironment implements ModelCache.
func (i *InMemoryModelCache) DelEnvironment(pid uint, eid uint, cluster string, namespace string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.dataMap, cacheKey(models.ResEnvironment, eid))
	delete(i.dataMap, envCacheKey(cluster, namespace))
	return nil
}

// UpsertVirtualSpace implements ModelCache.
func (i *InMemoryModelCache) UpsertVirtualSpace(vid uint, name string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	n := &Entity{Name: name, Kind: models.ResVirtualSpace, ID: vid}
	i.dataMap[n.cacheKey()] = n
	return nil
}

// DelVirtualSpace implements ModelCache.
func (i *InMemoryModelCache) DelVirtualSpace(vid uint) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.dataMap, cacheKey(models.ResVirtualSpace, vid))
	return nil
}

// UpsertProject implements ModelCache.
func (i *InMemoryModelCache) UpsertProject(tid uint, pid uint, name string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	n := &Entity{Name: name, Kind: models.ResProject, ID: pid, Owner: []*Entity{{Kind: models.ResTenant, ID: tid}}}
	i.dataMap[n.cacheKey()] = n
	return nil
}

// DelProject implements ModelCache.
func (i *InMemoryModelCache) DelProject(tid uint, pid uint) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.dataMap, cacheKey(models.ResProject, pid))
	return nil
}

// UpsertTenant implements ModelCache.
func (i *InMemoryModelCache) UpsertTenant(tid uint, name string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	n := &Entity{Name: name, Kind: models.ResTenant, ID: tid}
	i.dataMap[n.cacheKey()] = n
	return nil
}

// DelTenant implements ModelCache.
func (i *InMemoryModelCache) DelTenant(tid uint) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.dataMap, cacheKey(models.ResTenant, tid))
	return nil
}

func (i *InMemoryModelCache) get(key string) CommonResourceIface {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if val, ok := i.dataMap[key]; !ok {
		return nil
	} else {
		return val
	}
}

// FindParents implements ModelCache.
// NOTE: find parents recursively returns all parents AND IT SELF.
func (i *InMemoryModelCache) FindParents(kind string, id uint) []CommonResourceIface {
	return i.findAllParents(kind, id)
}

func (i *InMemoryModelCache) findAllParents(kind string, id uint) []CommonResourceIface {
	n := i.get(cacheKey(kind, id))
	if n == nil {
		return nil
	}
	// nolint: gomnd
	ret := make([]CommonResourceIface, 0, 2)
	ret = append(ret, n) // append self
	for _, owner := range n.GetOwners() {
		ret = append(ret, i.findAllParents(owner.GetKind(), owner.GetID())...)
	}
	return ret
}

// FindResource implements ModelCache.
func (i *InMemoryModelCache) FindResource(kind string, id uint) CommonResourceIface {
	return i.get(cacheKey(kind, id))
}

// GetUserAuthority implements ModelCache.
func (i *InMemoryModelCache) GetUserAuthority(user models.CommonUserIface) *UserAuthority {
	auth, ok := i.authorities.Get(userAuthorityKey(user.GetUsername()))
	if !ok {
		return i.FlushUserAuthority(user)
	}
	return auth
}

// FlushUserAuthority implements ModelCache.
func (c *InMemoryModelCache) FlushUserAuthority(user models.CommonUserIface) *UserAuthority {
	auth := new(UserAuthority)
	sysrole := models.SystemRole{ID: user.GetSystemRoleID()}
	if err := c.DB.First(&sysrole).Error; err != nil {
		log.Error(err, "faield to get user system role", "user", user.GetUsername())
	}
	var turs []models.TenantUserRels
	if err := c.DB.Preload("Tenant").Find(&turs, "user_id = ?", user.GetID()).Error; err != nil {
		log.Error(err, "faield to get user tenantlist", "user", user.GetUsername())
	}
	var purs []models.ProjectUserRels
	if err := c.DB.Preload("Project").Find(&purs, "user_id = ?", user.GetID()).Error; err != nil {
		log.Error(err, "faield to get user projectlist", "user", user.GetUsername())
	}
	var eurs []models.EnvironmentUserRels
	if err := c.DB.Preload("Environment").Find(&eurs, "user_id = ?", user.GetID()).Error; err != nil {
		log.Error(err, "faield to get user environmentlist", "user", user.GetUsername())
	}
	var vurs []models.VirtualSpaceUserRels
	if err := c.DB.Preload("VirtualSpace").Find(&vurs, "user_id = ?", user.GetID()).Error; err != nil {
		log.Error(err, "faield to get user virtualspacelist", "user", user.GetUsername())
	}
	auth.SystemRole = sysrole.RoleCode
	auth.Tenants = make([]*UserResource, len(turs))
	auth.Projects = make([]*UserResource, len(purs))
	auth.Environments = make([]*UserResource, len(eurs))
	auth.VirtualSpaces = make([]*UserResource, len(vurs))
	for i := range turs {
		auth.Tenants[i] = &UserResource{
			ID:      int(turs[i].TenantID),
			Name:    turs[i].Tenant.TenantName,
			Role:    turs[i].Role,
			IsAdmin: turs[i].Role == models.TenantRoleAdmin,
		}
	}
	for i := range purs {
		auth.Projects[i] = &UserResource{
			ID:      int(purs[i].ProjectID),
			Name:    purs[i].Project.ProjectName,
			Role:    purs[i].Role,
			IsAdmin: purs[i].Role == models.ProjectRoleAdmin,
		}
	}
	for i := range eurs {
		auth.Environments[i] = &UserResource{
			ID:      int(eurs[i].EnvironmentID),
			Name:    eurs[i].Environment.EnvironmentName,
			Role:    eurs[i].Role,
			IsAdmin: eurs[i].Role == models.EnvironmentRoleOperator,
		}
	}
	for i := range vurs {
		auth.VirtualSpaces[i] = &UserResource{
			ID:      int(vurs[i].VirtualSpaceID),
			Name:    vurs[i].VirtualSpace.VirtualSpaceName,
			Role:    vurs[i].Role,
			IsAdmin: vurs[i].Role == models.VirtualSpaceRoleAdmin,
		}
	}
	// update cache
	key := userAuthorityKey(user.GetUsername())
	c.authorities.Remove(key)
	c.authorities.Add(key, auth)
	return auth
}
