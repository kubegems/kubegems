package tenanthandler

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"kubegems.io/pkg/agent/apis/types"
)

type OversoldConfig struct {
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Storage float64 `json:"storage"`
}

func (h *TenantHandler) ValidateTenantResourceQuota(ctx context.Context, clusterOversold []byte, clustername string, origin, need v1.ResourceList) error {
	statistics := &types.ClusterResourceStatistics{}

	cli, err := h.GetAgents().ClientOf(ctx, clustername)
	if err != nil {
		return err
	}
	if err := cli.Extend().ClusterResourceStatistics(ctx, statistics); err != nil {
		return fmt.Errorf("验证资源错误 %w", err)
	}

	// 当前的限制
	originCPU := origin[v1.ResourceLimitsCPU]
	originMemory := origin[v1.ResourceLimitsMemory]
	originStorage := origin[v1.ResourceRequestsStorage]

	// 当前申请的资源量
	needCPU := need[v1.ResourceLimitsCPU]
	needMemory := need[v1.ResourceLimitsMemory]
	needStorage := need[v1.ResourceRequestsStorage]

	needCpuV := needCPU.Value() - originCPU.Value()
	needMemoryV := needMemory.Value() - originMemory.Value()
	needStorageV := needStorage.Value() - originStorage.Value()

	// 资源超分验证
	oversoldConfig := OversoldConfig{
		CPU:     1,
		Memory:  1,
		Storage: 1,
	}
	if len(clusterOversold) == 0 {
		clusterOversold = []byte("{}")
	}

	if err := json.Unmarshal(clusterOversold, &oversoldConfig); err != nil {
		return fmt.Errorf("资源超卖配置错误 %w", err)
	}

	// 已经分配了的
	allocatedCpu := statistics.TenantAllocated[v1.ResourceLimitsCPU]
	allocatedMemory := statistics.TenantAllocated[v1.ResourceLimitsMemory]
	allocatedStorage := statistics.TenantAllocated[v1.ResourceRequestsStorage]

	// 实际总量
	capacityCpu := statistics.Capacity[v1.ResourceCPU]
	capacityMemory := statistics.Capacity[v1.ResourceMemory]
	capacityStorage := statistics.Capacity[v1.ResourceEphemeralStorage]

	// 超分总量
	oCapacityCPU := int64(float64(capacityCpu.AsDec().UnscaledBig().Int64()) * oversoldConfig.CPU)
	oCapacityMemory := int64(float64(capacityMemory.AsDec().UnscaledBig().Int64()) * oversoldConfig.Memory)
	oCapacityStorage := int64(float64(capacityStorage.AsDec().UnscaledBig().Int64()) * oversoldConfig.Storage)

	leftCpu := oCapacityCPU - allocatedCpu.AsDec().UnscaledBig().Int64()
	leftMemory := oCapacityMemory - allocatedMemory.AsDec().UnscaledBig().Int64()
	leftStorage := oCapacityStorage - allocatedStorage.AsDec().UnscaledBig().Int64()

	errRes := []string{}
	if leftCpu < needCpuV {
		errRes = append(errRes, fmt.Sprintf("CPU(left %v but need %v)", leftCpu, needCpuV))
	}
	if leftMemory < needMemoryV {
		errRes = append(errRes, fmt.Sprintf("Memory(left %v but need %v)", leftMemory, needMemoryV))
	}
	if leftStorage < needStorageV {
		errRes = append(errRes, fmt.Sprintf("Storage(left %v but need %v)", leftStorage, needStorageV))
	}
	if len(errRes) > 0 {
		return fmt.Errorf("集群资源不足%v", errRes)
	}
	return nil
}
