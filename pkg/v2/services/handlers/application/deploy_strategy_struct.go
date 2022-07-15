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

package application

import (
	"encoding/json"

	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	istionetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
)

// ExtendCanaryStrategy 用于扩展原canary策略中 virtualservice 匹配的字段
type ExtendCanaryStrategy struct {
	rolloutsv1alpha1.CanaryStrategy
	// TrafficRouting hosts all the supported service meshes supported to enable more fine-grained traffic routing
	TrafficRouting *RolloutTrafficRouting `json:"trafficRouting,omitempty" protobuf:"bytes,4,opt,name=trafficRouting"`
}

func (s *ExtendCanaryStrategy) ToCanaryStrategy() *rolloutsv1alpha1.CanaryStrategy {
	if s == nil {
		return nil
	}
	content, _ := json.Marshal(s)
	canaryStrategy := &rolloutsv1alpha1.CanaryStrategy{}
	json.Unmarshal(content, canaryStrategy)
	return canaryStrategy
}

func ExtendCanaryStrategyFromCanaryStrategy(from *rolloutsv1alpha1.CanaryStrategy) *ExtendCanaryStrategy {
	if from == nil {
		return nil
	}
	s := &ExtendCanaryStrategy{}
	content, _ := json.Marshal(from)
	json.Unmarshal(content, s)
	return s
}

// RolloutTrafficRouting hosts all the different configuration for supported service meshes to enable more fine-grained traffic routing
type RolloutTrafficRouting struct {
	rolloutsv1alpha1.RolloutTrafficRouting
	// Istio holds Istio specific configuration to route traffic
	Istio *IstioTrafficRouting `json:"istio,omitempty" protobuf:"bytes,1,opt,name=istio"`
}

type IstioTrafficRouting struct {
	rolloutsv1alpha1.IstioTrafficRouting
	// VirtualService references an Istio VirtualService to modify to shape traffic
	VirtualService IstioVirtualService `json:"virtualService" protobuf:"bytes,1,opt,name=virtualService"`
}

// 根据header进行灰度
type IstioVirtualService struct {
	rolloutsv1alpha1.IstioVirtualService
	Uri           *istionetworkingv1alpha3.StringMatch            `json:"uri,omitempty"`
	Headers       map[string]*istionetworkingv1alpha3.StringMatch `json:"headers,omitempty"`
	IgnoreUriCase bool                                            `json:"ignoreUriCase,omitempty"`
}
