package storage

const (
	AnnotationInUse                                            = GroupName + "/in-use"
	AnnotationAllowSnapshot                                    = GroupName + "/allow-snapshot"
	AnnotationVolumeSnapshotAnnotationKeyPersistentVolumeClaim = GroupName + "/persistentvolumevlaim"

	AnnotationStorageProvisioner = "volume.beta.kubernetes.io/storage-provisioner"
)
