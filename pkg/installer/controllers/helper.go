package controllers

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func MarshalValues(vals map[string]interface{}) runtime.RawExtension {
	if vals == nil {
		return runtime.RawExtension{}
	}
	bytes, _ := json.Marshal(vals)
	return runtime.RawExtension{Raw: bytes}
}

func UnmarshalValues(val runtime.RawExtension) map[string]interface{} {
	if val.Raw == nil {
		return nil
	}
	var vals interface{}
	_ = yaml.Unmarshal(val.Raw, &vals)

	if kvs, ok := vals.(map[string]interface{}); ok {
		return kvs
	}
	if arr, ok := vals.([]interface{}); ok {
		// is format of --set K=V
		kvs := make(map[string]interface{}, len(arr))
		for _, kv := range arr {
			if kv, ok := kv.(map[string]interface{}); ok {
				for k, v := range kv {
					kvs[k] = v
				}
			}
		}
		return kvs
	}
	return nil
}
