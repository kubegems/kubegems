package models

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"kubegems.io/pkg/utils"
)

const (
	StatusVeryHigh = "very_high" // 非常高，要扩容
	StatusHigh     = "high"      // 高，要扩容
	StatusLow      = "low"       // 低，要缩容

	ColorYellow = "yellow"
	ColorRed    = "red"

	Ki = 1 << 10 // 1024
	Mi = 1 << 20
	Gi = 1 << 30

	resourceCPU    = "cpu"
	resourceMemory = "memory"

	minCPULimitCores    = 0.1      // 10m
	minMemoryLimitBytes = 100 * Mi // 100Mi
)

// Workload workload资源使用清单
type Workload struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	ClusterName       string
	Namespace         string
	Type              string
	Name              string
	CPULimitStdvar    float64
	MemoryLimitStdvar float64

	Containers []*Container `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`

	*Notice `gorm:"-"`
}

// Container 不注册为gorm model
type Container struct {
	ID uint `gorm:"primarykey"`

	Name    string
	PodName string

	CPULimitCore     float64
	MemoryLimitBytes int64 // 限制

	CPUUsageCore float64
	CPUPercent   float64 // 使用率

	MemoryUsageBytes float64
	MemoryPercent    float64 // 使用率

	WorkloadID uint
	Workload   *Workload `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type Condition struct {
	Pods         []string
	CPUStatus    *CPUMemoryStatus `json:"CPUStatus"`
	MemoryStatus *CPUMemoryStatus `json:"MemoryStatus"`

	MaxCPUContainer    *Container `json:"-"`
	MaxMemoryContainer *Container `json:"-"`
	CPULimitCore       float64    `json:"-"`
	MemoryLimitBytes   int64      `json:"-"`
}

type CPUMemoryStatus struct {
	CurrentRate  float64
	CurrentLimit string // 带单位

	Status string // 扩容或缩容

	SuggestLimit    string // 带单位
	SuggestMinLimit string // 带单位
	SuggestMaxLimit string // 带单位
}

func (status *CPUMemoryStatus) AddSuggest(SuggestType string, currentUsage float64) *CPUMemoryStatus {
	switch {
	case status.CurrentRate < 0.1:
		// 使用率低于10%, 且limit小于最小值的，忽略
		if (SuggestType == resourceCPU && currentUsage/0.3 < minCPULimitCores) ||
			(SuggestType == resourceMemory && currentUsage/0.3 < minMemoryLimitBytes) {
			return nil
		}
		status.Status = StatusLow
	case status.CurrentRate > 0.9:
		status.Status = StatusVeryHigh
	case status.CurrentRate > 0.6:
		status.Status = StatusHigh
	default:
		return nil
	}

	switch SuggestType {
	case resourceCPU:
		// 使得使用率在40% ~ [30%, 50%]
		status.SuggestLimit = resource.NewMilliQuantity(int64((currentUsage*1000)/0.4), resource.DecimalSI).String()
		status.SuggestMinLimit = resource.NewMilliQuantity(int64((currentUsage*1000)/0.5), resource.DecimalSI).String()
		status.SuggestMaxLimit = resource.NewMilliQuantity(int64((currentUsage*1000)/0.3), resource.DecimalSI).String()
	case resourceMemory:
		// 使得使用率在40% ~ [30%, 50%]
		status.SuggestLimit = resource.NewQuantity(convertBytes(currentUsage/0.4), resource.BinarySI).String()
		status.SuggestMinLimit = resource.NewQuantity(convertBytes(currentUsage/0.5), resource.BinarySI).String()
		status.SuggestMaxLimit = resource.NewQuantity(convertBytes(currentUsage/0.3), resource.BinarySI).String()
	default:
		return nil
	}
	return status
}

// 对单位Mi取整
func convertBytes(bytes float64) int64 {
	switch {
	case bytes/Gi > 10:
		return (int64(bytes) / Gi) * Gi
	case bytes/Mi > 10:
		return (int64(bytes) / Mi) * Mi
	case bytes/Ki > 10:
		return (int64(bytes) / Ki) * Ki
	default:
		return int64(bytes)
	}
}

type Notice struct {
	Color      string               // eg, yellow
	Conditions map[string]Condition // 按容器名分组
}

func (w *Workload) AddNotice() {
	notice := &Notice{
		Color:      ColorYellow,
		Conditions: map[string]Condition{},
	}

	for _, c := range w.Containers {
		if c.CPUPercent > 0.9 || c.MemoryPercent > 0.9 {
			notice.Color = ColorRed
		}

		cond, ok := notice.Conditions[c.Name]
		if !ok {
			cond = Condition{
				MaxCPUContainer:    &Container{},
				MaxMemoryContainer: &Container{},
			}
		}
		if c.CPUPercent > cond.MaxCPUContainer.CPUPercent {
			cond.MaxCPUContainer = c
		}
		if c.MemoryPercent > cond.MaxMemoryContainer.MemoryPercent {
			cond.MaxMemoryContainer = c
		}

		cond.Pods = append(cond.Pods, c.PodName)
		if c.CPULimitCore != 0 {
			cond.CPULimitCore = c.CPULimitCore
			cond.MemoryLimitBytes = c.MemoryLimitBytes
		}
		notice.Conditions[c.Name] = cond
	}

	for condKey := range notice.Conditions {
		cond := notice.Conditions[condKey]
		if cond.MaxCPUContainer.Name != "" {
			cond.CPUStatus = (&CPUMemoryStatus{
				CurrentRate:  utils.RoundTo(cond.MaxCPUContainer.CPUPercent, 3),
				CurrentLimit: resource.NewMilliQuantity(int64(cond.CPULimitCore*1000), resource.DecimalSI).String(),
			}).AddSuggest(resourceCPU, cond.MaxCPUContainer.CPUUsageCore)
		}

		if cond.MaxMemoryContainer.Name != "" {
			cond.MemoryStatus = (&CPUMemoryStatus{
				CurrentRate:  utils.RoundTo(cond.MaxMemoryContainer.MemoryPercent, 3),
				CurrentLimit: resource.NewQuantity(int64(cond.MemoryLimitBytes), resource.BinarySI).String(),
			}).AddSuggest(resourceMemory, cond.MaxMemoryContainer.MemoryUsageBytes)
		}

		// 目前只会在小于最小limit时出现
		if cond.CPUStatus == nil && cond.MemoryStatus == nil {
			delete(notice.Conditions, condKey)
		} else {
			notice.Conditions[condKey] = cond
		}
	}

	w.Notice = notice
}

func (w *Workload) UniqueKey() string {
	return fmt.Sprintf("%s_%s_%s", w.Namespace, w.Type, w.Name)
}
