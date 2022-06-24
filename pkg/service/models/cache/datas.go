package cache

import (
	"encoding/json"
	"fmt"

	"kubegems.io/kubegems/pkg/service/models"
)

type CommonResourceIface interface {
	GetKind() string
	GetID() uint
	GetTenantID() uint
	GetProjectID() uint
	GetEnvironmentID() uint
	GetVirtualSpaceID() uint
	GetName() string
	GetCluster() string
	GetNamespace() string
	GetOwners() []CommonResourceIface
}

type Entity struct {
	Name      string    `json:",omitempty"`
	Kind      string    `json:",omitempty"`
	ID        uint      `json:",omitempty"`
	Namespace string    `json:",omitempty"`
	Cluster   string    `json:",omitempty"`
	Owner     []*Entity `json:",omitempty"`
	Children  []string  `json:",omitempty"`
}

func (n *Entity) MarshalBinary() ([]byte, error) {
	return json.Marshal(*n)
}

func (n *Entity) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, n)
}

func (n *Entity) cacheKey() string {
	return fmt.Sprintf("%s_%v", n.Kind, n.ID)
}

func (n *Entity) toPair() map[string]interface{} {
	return map[string]interface{}{
		n.cacheKey(): n,
	}
}

func (n *Entity) toEnvPair() map[string]interface{} {
	if n.Kind != models.ResEnvironment {
		return nil
	}
	return map[string]interface{}{
		envCacheKey(n.Cluster, n.Namespace): n,
	}
}

func (n *Entity) GetKind() string {
	return n.Kind
}

func (n *Entity) GetID() uint {
	return n.ID
}

func (n *Entity) GetTenantID() uint {
	return n.getKindID(models.ResTenant)
}

func (n *Entity) GetProjectID() uint {
	return n.getKindID(models.ResProject)
}

func (n *Entity) GetEnvironmentID() uint {
	return n.getKindID(models.ResEnvironment)

}

func (n *Entity) GetVirtualSpaceID() uint {
	return n.getKindID(models.ResVirtualSpace)
}

func (n *Entity) GetName() string {
	return n.Name
}

func (n *Entity) GetCluster() string {
	return n.Cluster
}

func (n *Entity) GetNamespace() string {
	return n.Namespace
}

func (n *Entity) GetOwners() []CommonResourceIface {
	length := len(n.Owner)
	if length == 0 {
		return nil
	}
	ret := make([]CommonResourceIface, length)
	for i := 0; i < length; i++ {
		ret[i] = n.Owner[i]
	}
	return ret
}

func (n *Entity) getKindID(k string) uint {
	if k == n.Kind {
		return n.ID
	}
	return 0
}

func cacheKey(kind string, id uint) string {
	return fmt.Sprintf("%s_%v", kind, id)
}

func envCacheKey(cluster, namespace string) string {
	return fmt.Sprintf("env_%s_%s", cluster, namespace)
}

const FindParentScript = `
local cachekey = KEYS[1]
local kind = KEYS[2]
local id = KEYS[3]
local ret = {}
local function getparents(kind, id)
    local key = kind.."_"..id
    local current = redis.call("HGET", cachekey, key)
    if not current then
        return
    end
    table.insert(ret, current)
    local cdata = cjson.decode(current)
    if cdata["Owner"] then
        for k, parent in ipairs(cdata["Owner"]) do
            getparents(parent["Kind"], parent["ID"])
        end 
    end
end

local function reverse(arr)
    local nret = {}
    for k, it in ipairs(arr) do
        nret[#arr-k+1] = it
    end 
    return nret
end

getparents(kind, id)
return reverse(ret)
`
