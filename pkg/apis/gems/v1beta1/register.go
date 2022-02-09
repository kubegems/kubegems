package v1beta1

import (
	"github.com/kubegems/gems/pkg/apis/gems"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// SchemeGroupVersion is group version used to register these objects.
var SchemeGroupVersion = schema.GroupVersion{Group: gems.GroupName, Version: "v1beta1"}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

var (
	GroupVersion = SchemeGroupVersion
	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

var (
	SchemeTenant              = GroupVersion.WithKind("Tenant")
	SchemeTenantResourceQuota = GroupVersion.WithKind("TenantResourceQuota")
	SchemeTenantNetworkPolicy = GroupVersion.WithKind("TenantNetworkPolicy")
	SchemeTenantGateway       = GroupVersion.WithKind("TenantGateway")
	SchemeEnvironment         = GroupVersion.WithKind("Environment")
	SchemeResourceQuota       = GroupVersion.WithKind("ResourceQuota")
)
