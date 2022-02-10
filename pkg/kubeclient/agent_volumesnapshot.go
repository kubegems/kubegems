package kubeclient

import (
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
)

var (
	vshotGVKi = v1.SchemeGroupVersion.WithKind("VolumeSnapshot")
	vshotGVK  = &vshotGVKi
)

func (k KubeClient) GetVolumeSnapshot(cluster, namespace, name string, labels map[string]string) (*v1.VolumeSnapshot, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.VolumeSnapshot{}
	err = agentClient.GetObject(vshotGVK, obj, &namespace, &name)
	return obj, err
}

func (k KubeClient) GetVolumeSnapshotList(cluster, namespace string, labelSet map[string]string) (*[]v1.VolumeSnapshot, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.VolumeSnapshot{}
	err = agentClient.GetObjectList(vshotGVK, &list, &namespace, labelSet)
	return &list, err
}

func (k KubeClient) DeleteVolumeSnapshot(cluster, namespace, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(vshotGVK, &namespace, &name)
}

func (k KubeClient) CreateVolumeSnapshot(cluster, namespace, name string, data *v1.VolumeSnapshot) (*v1.VolumeSnapshot, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.CreateObject(vshotGVK, data, &namespace, &name)
	return data, err
}

func (k KubeClient) PatchVolumeSnapshot(cluster, namespace, name string, data *v1.VolumeSnapshot) (*v1.VolumeSnapshot, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	data.SetNamespace(namespace)
	err = agentClient.PatchObject(vshotGVK, data, &namespace, &name)
	return data, err
}
