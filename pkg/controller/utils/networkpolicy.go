package utils

import (
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/apis/gems"
)

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

func validNetworkPolicy(np *netv1.NetworkPolicy) bool {
	if len(np.Spec.Ingress) < 3 {
		return false
	}
	if len(np.Spec.Ingress[2].From) == 0 {
		return false
	}
	return true
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

func hasKindLabel(netpol *netv1.NetworkPolicy, kind string) bool {
	for _, exp := range netpol.Spec.Ingress[2].From[0].NamespaceSelector.MatchExpressions {
		if exp.Key == kind {
			return true
		}
	}
	return false
}