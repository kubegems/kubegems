package kubeclient

import (
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
)

var (
	vshotClassGVKi = v1.SchemeGroupVersion.WithKind("VolumeSnapshotClass")
	vshotClassGVK  = &vshotClassGVKi
)

func (k KubeClient) GetVolumeSnapshotClass(cluster, name string, _ map[string]string) (*v1.VolumeSnapshotClass, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	obj := &v1.VolumeSnapshotClass{}
	err = agentClient.GetObject(vshotClassGVK, obj, nil, &name)
	return obj, err
}

func (k KubeClient) GetVolumeSnapshotClassList(cluster string, labelSet map[string]string) (*[]v1.VolumeSnapshotClass, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	list := []v1.VolumeSnapshotClass{}
	err = agentClient.GetObjectList(vshotClassGVK, &list, nil, labelSet)
	return &list, err
}

func (k KubeClient) DeleteVolumeSnapshotClass(cluster, name string) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	return agentClient.DeleteObject(vshotClassGVK, nil, &name)
}

func (k KubeClient) CreateVolumeSnapshotClass(cluster, name string, data *v1.VolumeSnapshotClass) (*v1.VolumeSnapshotClass, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.CreateObject(vshotClassGVK, data, nil, &name)
	return data, err
}

func (k KubeClient) PatchVolumeSnapshotClass(cluster, name string, data *v1.VolumeSnapshotClass) (*v1.VolumeSnapshotClass, error) {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return nil, err
	}
	data.SetName(name)
	err = agentClient.PatchObject(vshotClassGVK, data, nil, &name)
	return data, err
}
