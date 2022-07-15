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

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/gems"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// event reason
const (
	ReasonFailedCreateSubResource = "FailedCreateSubResource"
	ReasonFailedCreate            = "FailedCreate"
	ReasonFailedDelete            = "FailedDelete"
	ReasonFailedUpdate            = "FailedUpdate"
	ReasonCreated                 = "Created"
	ReasonDeleted                 = "Deleted"
	ReasonUpdated                 = "Updated"
	ReasonUnknowError             = "UnknowError"
)

// owner
func ExistOwnerRef(meta metav1.ObjectMeta, owner metav1.OwnerReference) bool {
	var exist bool
	for _, ref := range meta.OwnerReferences {
		if ref.APIVersion == owner.APIVersion && ref.Kind == owner.Kind && ref.Name == owner.Name {
			exist = true
			break
		}
	}
	return exist
}

// only cpu and memory
func HasDifferentResources(origin, newone corev1.ResourceRequirements) bool {
	return !(origin.Requests.Cpu().Equal(newone.Requests.Cpu().DeepCopy()) &&
		origin.Requests.Memory().Equal(newone.Requests.Memory().DeepCopy()) &&
		origin.Limits.Cpu().Equal(newone.Limits.Cpu().DeepCopy()) &&
		origin.Limits.Memory().Equal(newone.Limits.Memory().DeepCopy()))
}

func GetCIDRs(c client.Client) ([]string, error) {
	ctx := context.Background()
	ret := []string{}
	nodeList := &corev1.NodeList{}
	if err := c.List(ctx, nodeList, &client.ListOptions{}); err != nil {
		return ret, err
	}
	for _, node := range nodeList.Items {
		ret = append(ret, node.Spec.PodCIDRs...)
	}
	return ret, nil
}

// network policy
func DefaultNetworkPolicy(namespace, name string, cidrs []string) netv1.NetworkPolicy {
	np := netv1.NetworkPolicy{}
	np.Name = name
	np.Namespace = namespace
	np.Spec = netv1.NetworkPolicySpec{
		PolicyTypes: []netv1.PolicyType{netv1.PolicyTypeIngress},
		Ingress: []netv1.NetworkPolicyIngressRule{
			{
				From: []netv1.NetworkPolicyPeer{
					{
						IPBlock: &netv1.IPBlock{
							CIDR:   "0.0.0.0/0",
							Except: cidrs,
						},
					},
				},
			},
			{
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      gems.LabelPlugins,
									Operator: metav1.LabelSelectorOpExists,
								},
							},
						},
					},
				},
			},
			{
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{},
						},
					},
				},
			},
		},
	}
	return np
}

func DelNamespaceSelector(np *netv1.NetworkPolicy, kind string) {
	if !validNetworkPolicy(np) {
		return
	}
	index := -1
	origin := np.Spec.Ingress[2].From[0].NamespaceSelector.MatchExpressions
	for idx, item := range origin {
		if item.Key == kind {
			index = idx
		}
	}
	if index != -1 {
		np.Spec.Ingress[2].From[0].NamespaceSelector.MatchExpressions = append(origin[:index], origin[index+1:]...)
	}
}

func validNetworkPolicy(np *netv1.NetworkPolicy) bool {
	if len(np.Spec.Ingress) < 3 {
		return false
	}
	if len(np.Spec.Ingress[2].From) == 0 {
		return false
	}
	return true
}

func hasKindLabel(netpol *netv1.NetworkPolicy, kind string) bool {
	for _, exp := range netpol.Spec.Ingress[2].From[0].NamespaceSelector.MatchExpressions {
		if exp.Key == kind {
			return true
		}
	}
	return false
}

func AddNamespaceSelector(np *netv1.NetworkPolicy, kind, value string) {
	if !validNetworkPolicy(np) {
		return
	}
	if hasKindLabel(np, kind) {
		return
	}
	sel := metav1.LabelSelectorRequirement{Key: kind, Operator: metav1.LabelSelectorOpIn, Values: []string{value}}
	np.Spec.Ingress[2].From[0].NamespaceSelector.MatchExpressions = append(np.Spec.Ingress[2].From[0].NamespaceSelector.MatchExpressions, sel)
}
