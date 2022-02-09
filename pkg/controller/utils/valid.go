package utils

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

func IsLimitRangeInvalid(limitRangeItems []v1.LimitRangeItem) ([]string, bool) {
	var (
		errmsg  []string
		invalid bool
	)
	for _, item := range limitRangeItems {
		for k, v := range item.DefaultRequest {
			if limitv, exist := item.Default[k]; exist {
				if v.Cmp(limitv) == 1 {
					l, _ := limitv.MarshalJSON()
					r, _ := v.MarshalJSON()
					msg := fmt.Sprintf("limitType %v error: %v limit value %v, requests value %v", item.Type, k, string(l), string(r))
					errmsg = append(errmsg, msg)
					invalid = true
				}
			}
		}
	}
	return errmsg, invalid
}
