// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhooks

import (
	"fmt"
	"strconv"

	networkingv1 "k8s.io/api/networking/v1"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
)

func IsGatewayHTTP2(tg gemsv1beta1.TenantGateway) bool {
	if tg.Spec.ConfigMapData != nil {
		isHttp2, _ := strconv.ParseBool(tg.Spec.ConfigMapData["http2"])
		return isHttp2
	}
	return false
}

func IsIngressGRPC(ingress networkingv1.Ingress) bool {
	if ingress.Annotations != nil {
		_, ok := ingress.Annotations["nginx.org/grpc-services"]
		return ok
	}
	return false
}

func CheckGatewayAndIngressProtocol(tg gemsv1beta1.TenantGateway, ingresses []networkingv1.Ingress) error {
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
