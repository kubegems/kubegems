package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// ResourceEnough 资源 是否足够，不够给出不够的错误项
func ResourceEnough(total, used, need corev1.ResourceList) (bool, []string) {
	valid := true
	errmsgs := []string{}

	for k, needv := range need {
		totalv, totalExist := total[k]
		usedv, usedExist := used[k]
		if !totalExist || !usedExist {
			continue
		}
		totalv.Sub(usedv)
		if totalv.Cmp(needv) == -1 {
			valid = false
			left := totalv.DeepCopy()
			needv := needv.DeepCopy()
			msg := fmt.Sprintf("%v left %v but need %v", k.String(), left.String(), needv.String())
			errmsgs = append(errmsgs, msg)
		}
	}
	return valid, errmsgs
}

// SubResource 用新的值去减去旧的，得到差
func SubResource(oldres, newres corev1.ResourceList) corev1.ResourceList {
	retres := corev1.ResourceList{}
	for k, v := range newres {
		ov, exist := oldres[k]
		if exist {
			v.Sub(ov)
			retres[k] = v
		} else {
			retres[k] = v
		}
	}
	return retres
}

func ResourceIsEnough(total, used, need corev1.ResourceList, resources []corev1.ResourceName) (bool, []string) {
	ret := true
	msgs := []string{}
	for _, resource := range resources {
		totalv := total[resource]
		usedv := used[resource]
		needv := need[resource]
		if needv.IsZero() {
			continue
		}
		tmp := totalv.DeepCopy()
		tmp.Sub(usedv)
		if tmp.Cmp(needv) == -1 {
			l, _ := tmp.MarshalJSON()
			n, _ := needv.MarshalJSON()
			msg := fmt.Sprintf("%s not enough to apply, tenant left %s but need %s", resource, string(l), string(n))
			msgs = append(msgs, msg)
			ret = false
		}
	}
	return ret, msgs
}

// only cpu and memory
func HasDifferentResources(origin, newone corev1.ResourceRequirements) bool {
	return !(origin.Requests.Cpu().Equal(newone.Requests.Cpu().DeepCopy()) &&
		origin.Requests.Memory().Equal(newone.Requests.Memory().DeepCopy()) &&
		origin.Limits.Cpu().Equal(newone.Limits.Cpu().DeepCopy()) &&
		origin.Limits.Memory().Equal(newone.Limits.Memory().DeepCopy()))
}
