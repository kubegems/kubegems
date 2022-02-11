/*
mutate:

	1. Tenant: 				无
	2. TenantResourceQuota: create/update 填充默认值
	3. TenantNetworkPolicy: 无
	4. TenantGateway: 		create/update 填充replicas与type
	5. Environment： 		create/update 填充默认的ResourceQuota,填充默认的LimitRange

validate:

	1. Tenant:				update 禁止更新租户名字, 删除前，如果存在环境，则不允许删除
	2. TenantResourceQuota: 创建和更新，禁止超过当前集群容量，禁止删除
	3. TenantNetworkPolicy: create,禁止创建属于不存在的租户NPOL, update禁止删除租户label, delete禁止删除
	4. TenantGateway: 		create/update禁止创建属于不存在的租户GATEWAY，禁止无IngressClass
	5. Environment： 		create/update 验证资源是超过限制，验证limigrange是否合法
	6. Namespace:			delete 禁止删除/属于环境的namespace
*/

// m1
//+kubebuilder:webhook:verbs=create;update,path=/mutate,mutating=true,failurePolicy=fail,groups=gems.kubegems.io,resources=tenantresourcequotas,versions=v1beta1,name=mutate.resourcequota.dev,sideEffects=None,admissionReviewVersions=v1

// m2
//+kubebuilder:webhook:path=/mutate,mutating=true,failurePolicy=fail,groups=gems.kubegems.io,resources=environments,verbs=create;update,versions=v1beta1,name=mutate.environment.dev,sideEffects=None,admissionReviewVersions=v1

// m3
//+kubebuilder:webhook:path=/mutate,mutating=true,failurePolicy=fail,groups=gems.kubegems.io,resources=tenantgateways,verbs=create;update,versions=v1beta1,name=mutate.gateway.dev,sideEffects=None,admissionReviewVersions=v1

// m4
//+kubebuilder:webhook:path=/mutate,mutating=true,failurePolicy=fail,groups=extensions,resources=ingresses,verbs=create;update,versions=v1beta1,name=mutate.ingress.dev,sideEffects=None,admissionReviewVersions=v1

// v1
//+kubebuilder:webhook:verbs=update,path=/validate,mutating=false,failurePolicy=fail,groups=gems.kubegems.io,resources=tenants,versions=v1beta1,name=validate.tenant.dev,sideEffects=None,admissionReviewVersions=v1

// v2
//+kubebuilder:webhook:verbs=create;update;delete,path=/validate,mutating=false,failurePolicy=fail,groups=gems.kubegems.io,resources=tenantresourcequotas,versions=v1beta1,name=validate.tenantresourcequota.dev,sideEffects=None,admissionReviewVersions=v1

// v3
//+kubebuilder:webhook:verbs=create;update;delete,path=/validate,mutating=false,failurePolicy=fail,groups=gems.kubegems.io,resources=tenantnetworkpolicies,versions=v1beta1,name=validate.tenantnetworkpolicy.dev,sideEffects=None,admissionReviewVersions=v1

// v4
//+kubebuilder:webhook:verbs=create;update;delete,path=/validate,mutating=false,failurePolicy=fail,groups=gems.kubegems.io,resources=tenantgateways,versions=v1beta1,name=validate.tenantgateway.dev,sideEffects=None,admissionReviewVersions=v1

// v5
//+kubebuilder:webhook:verbs=create;update,path=/validate,mutating=false,failurePolicy=fail,groups=gems.kubegems.io,resources=environments,versions=v1beta1,name=validate.environment.dev,sideEffects=None,admissionReviewVersions=v1

// v6
//+kubebuilder:webhook:path=/validate,mutating=false,failurePolicy=fail,groups="",resources=namespaces,verbs=delete,versions=*,name=valiate.namespace.dev,sideEffects=None,admissionReviewVersions=v1

// v6
//+kubebuilder:webhook:verbs=create;update,path=/validate,mutating=false,failurePolicy=fail,groups=networking.istio.io,resources=gateways,versions=v1beta1,name=validate.istiogateway.dev,sideEffects=None,admissionReviewVersions=v1

// 所有的环境下的资源，需要注入label
//+kubebuilder:webhook:path=/label-injector,mutating=true,failurePolicy=ignore,groups="",resources=pods;configmaps;secrets;services;daemonsets;deployments;statefulsets;jobs;cronjobs;persistentvolumeclaims,verbs=create;update,versions=*,name=mutate.label-injector.dev,sideEffects=None,admissionReviewVersions=v1

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete

package webhooks
