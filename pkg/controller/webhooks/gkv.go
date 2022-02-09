package webhooks

import (
	gemsv1beta1 "github.com/kubegems/gems/pkg/apis/gems/v1beta1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	gkvTenant = metav1.GroupVersionKind{
		Group:   gemsv1beta1.GroupVersion.Group,
		Version: gemsv1beta1.GroupVersion.Version,
		Kind:    "Tenant",
	}
	gkvTenantResourceQuota = metav1.GroupVersionKind{
		Group:   gemsv1beta1.GroupVersion.Group,
		Version: gemsv1beta1.GroupVersion.Version,
		Kind:    "TenantResourceQuota",
	}
	gkvTenantNetworkPolicy = metav1.GroupVersionKind{
		Group:   gemsv1beta1.GroupVersion.Group,
		Version: gemsv1beta1.GroupVersion.Version,
		Kind:    "TenantNetworkPolicy",
	}
	gkvTenantGateway = metav1.GroupVersionKind{
		Group:   gemsv1beta1.GroupVersion.Group,
		Version: gemsv1beta1.GroupVersion.Version,
		Kind:    "TenantGateway",
	}
	gkvEnvironment = metav1.GroupVersionKind{
		Group:   gemsv1beta1.GroupVersion.Group,
		Version: gemsv1beta1.GroupVersion.Version,
		Kind:    "Environment",
	}
	gkvResourceQuota = metav1.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "ResourceQuota",
	}
	gkvNamespace = metav1.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Namespace",
	}
	gkvConfigMap = metav1.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "ConfigMap",
	}
	gkvSecret = metav1.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Secret",
	}
	gkvService = metav1.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Service",
	}
	gkvPvc = metav1.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "PersistentVolumeClaim",
	}
	gkvJob = metav1.GroupVersionKind{
		Group:   batchv1.SchemeGroupVersion.Group,
		Version: batchv1.SchemeGroupVersion.Version,
		Kind:    "Job",
	}
	gkvCronJob = metav1.GroupVersionKind{
		Group:   batchv1beta1.SchemeGroupVersion.Group,
		Version: batchv1beta1.SchemeGroupVersion.Version,
		Kind:    "CronJob",
	}
	gkvDeployment = metav1.GroupVersionKind{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: batchv1beta1.SchemeGroupVersion.Version,
		Kind:    "Deployment",
	}
	gkvDaemonSet = metav1.GroupVersionKind{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: batchv1beta1.SchemeGroupVersion.Version,
		Kind:    "Daemonset",
	}
	gkvStatefulSet = metav1.GroupVersionKind{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: batchv1beta1.SchemeGroupVersion.Version,
		Kind:    "StatefulSet",
	}
	gkvIngress = metav1.GroupVersionKind{
		Group:   extv1beta1.SchemeGroupVersion.Group,
		Version: extv1beta1.SchemeGroupVersion.Version,
		Kind:    "Ingress",
	}
	gvkIstioGateway = metav1.GroupVersionKind{
		Group:   istiov1beta1.SchemeGroupVersion.Group,
		Version: istiov1beta1.SchemeGroupVersion.Version,
		Kind:    "Gateway",
	}
)
