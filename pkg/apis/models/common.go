package models

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

type Properties map[string]interface{}

func (p Properties) ToRawExtension() *runtime.RawExtension {
	raw, _ := json.Marshal(p)
	return &runtime.RawExtension{Raw: raw}
}
