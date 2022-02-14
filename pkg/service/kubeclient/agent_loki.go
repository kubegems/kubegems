package kubeclient

import (
	"net/http"

	"kubegems.io/pkg/utils/loki"
)

type QueryResponseStruct struct {
	Message   string
	Data      *loki.QueryResponseData
	ErrorData interface{}
}

type LabelResponseStruct struct {
	Message   string
	Data      []string
	ErrorData interface{}
}

func (k *KubeClient) QueryRange(cluster string, query map[string]string) (*loki.QueryResponseData, error) {
	url := formatURL(nil, nil, query, "/custom/loki/v1/queryrange")
	ret := &loki.QueryResponseData{}
	if err := k.request(http.MethodGet, cluster, url, nil, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (k *KubeClient) Labels(cluster string, query map[string]string) ([]string, error) {
	url := formatURL(nil, nil, query, "/custom/loki/v1/labels")
	ret := []string{}
	if err := k.request(http.MethodGet, cluster, url, nil, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (k *KubeClient) LabelValues(cluster string, label string, query map[string]string) ([]string, error) {
	query["label"] = label
	url := formatURL(nil, nil, query, "/custom/loki/v1/labelvalues")
	ret := []string{}
	if err := k.request(http.MethodGet, cluster, url, nil, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (k *KubeClient) Series(cluster string, query map[string]string) (interface{}, error) {
	url := formatURL(nil, nil, query, "/custom/loki/v1/series")
	ret := []interface{}{}
	if err := k.request(http.MethodGet, cluster, url, nil, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}
