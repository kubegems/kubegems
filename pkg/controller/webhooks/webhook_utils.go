package webhooks

import (
	"fmt"
	"strconv"

	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	gemsv1beta1 "kubegems.io/pkg/apis/gems/v1beta1"
)

func IsGatewayHTTP2(tg gemsv1beta1.TenantGateway) bool {
	if tg.Spec.ConfigMapData != nil {
		isHttp2, _ := strconv.ParseBool(tg.Spec.ConfigMapData["http2"])
		return isHttp2
	}
	return false
}

func IsIngressGRPC(ingress ext_v1beta1.Ingress) bool {
	if ingress.Annotations != nil {
		_, ok := ingress.Annotations["nginx.org/grpc-services"]
		return ok
	}
	return false
}

func CheckGatewayAndIngressProtocol(tg gemsv1beta1.TenantGateway, ingresses []ext_v1beta1.Ingress) error {
	if !IsGatewayHTTP2(tg) {
		for _, ingress := range ingresses {
			if IsIngressGRPC(ingress) {
				return fmt.Errorf("ingress [%s] services [%s] use grpc protocol, the gateway [%s] must config http/2",
					ingress.Name, ingress.Annotations["nginx.org/grpc-services"], tg.Name)
			}
		}
	}
	return nil
}
