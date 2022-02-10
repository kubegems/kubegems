package msgbus

import (
	"fmt"
	"strings"
	"time"
)

type MessageType string

const (
	Approve MessageType = "approve"       // 审批
	Message MessageType = "message"       // 消息
	Changed MessageType = "objectChanged" // k8s 对象变动
	Alert   MessageType = "alert"         // 告警消息
)

type EventKind string

const (
	Add    EventKind = "add"
	Update EventKind = "update"
	Delete EventKind = "delete"
)

type ResourceType string

const (
	VirtualSpace ResourceType = "virtualSpace"
	Tenant       ResourceType = "tenant"
	Project      ResourceType = "project"
	Environment  ResourceType = "environment"
	Application  ResourceType = "application"
	Cluster      ResourceType = "cluster"
	User         ResourceType = "user"

	TenantResourceQuota ResourceType = "tenant-resource-quota"
)

type InvolvedObject struct {
	Cluster        string
	Group          string
	Kind           string
	Version        string
	NamespacedName string
}

func NamespacedNameFrom(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// default/a   -> default a
// default     -> default
// ""			-> "",""
// a/b/c        -> a,b
// NamespacedNameSplit
func NamespacedNameSplit(nm string) (string, string) {
	arr := strings.Split(nm, "/")
	switch len(arr) {
	case 0:
		return "", ""
	case 1:
		return "", arr[0]
	default:
		return arr[0], arr[1]
	}
}

type NotifyMessage struct {
	MessageType
	EventKind
	InvolvedObject *InvolvedObject
	Content        interface{}
}

type MessageContent struct {
	ResourceType
	ResouceID     uint
	CreatedAt     time.Time
	From          string
	To            []uint // 只在通知消息时有
	Detail        string
	AffectedUsers []uint // 受影响的用户
}

type CurrentWatch map[string]map[string][]string // cluster->kind->[]namespacedName

type ControlMessage struct {
	MessageType `json:"kind"`
	Content     CurrentWatch
}

type MessageTarget struct {
	Message NotifyMessage
	Users   []uint
}
