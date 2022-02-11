package networking

const (
	AnnotationVirtualDomain = GroupName + "/virtualdomain"
	AnnotationVirtualSpace  = GroupName + "/virtualspace"
	AnnotationIstioGateway  = GroupName + "/istioGateway"

	LabelIngressClass = GroupName + "/ingressClass" // ingress打标签用以筛选
)
