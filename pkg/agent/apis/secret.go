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

package apis

import (
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/utils/certificate"
	"kubegems.io/library/rest/response"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SecretHandler struct {
	C       client.Client
	cluster cluster.Interface
}

type SecretWithCertsInfo struct {
	Secret   *corev1.Secret                     `json:"secret,omitempty"`
	CertInfo map[string]certificate.Certificate `json:"certInfo,omitempty"`
}

func (s SecretWithCertsInfo) GetName() string {
	return s.Secret.GetName()
}

func (s SecretWithCertsInfo) GetCreationTimestamp() metav1.Time {
	return s.Secret.CreationTimestamp
}

// @Tags			Agent.V1
// @Summary		获取Secret列表数据
// @Description	获取Secret列表数据,其中包含了对 tls 类型的secret证书详情
// @Accept			json
// @Produce		json
// @Param			order		query		string																					false	"page"
// @Param			search		query		string																					false	"search"
// @Param			page		query		int																						false	"page"
// @Param			size		query		int																						false	"page"
// @Param			namespace	path		string																					true	"namespace"
// @Param			cluster		path		string																					true	"cluster"
// @Success		200			{object}	handlers.ResponseStruct{Data=response.Page[corev1.Secret]{List=[]SecretWithCertsInfo}}	"Secrets"
// @Router			/v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/secrets [get]
// @Security		JWT
func (h *SecretHandler) List(c *gin.Context) {
	ns := c.Param("namespace")
	if ns == "_all" || ns == "_" {
		ns = ""
	}
	list := &corev1.SecretList{}
	listOptions := &client.ListOptions{
		Namespace:     ns,
		LabelSelector: getLabelSelector(c),
	}
	if err := h.C.List(c.Request.Context(), list, listOptions); err != nil {
		NotOK(c, err)
		return
	}

	listWithCertsInfo := make([]SecretWithCertsInfo, len(list.Items))
	for i, secret := range list.Items {
		listWithCertsInfo[i] = SecretWithCertsInfo{
			Secret:   &list.Items[i],
			CertInfo: parseCertsInfo(secret),
		}
	}
	pageData := response.PageObjectFromRequest(c.Request, listWithCertsInfo)
	if iswatch, _ := strconv.ParseBool(c.Query("watch")); iswatch {
		// list
		c.SSEvent("data", pageData)
		c.Writer.Flush()
		// watch
		WatchEvents(c, h.cluster, list, listOptions)
		return
	} else {
		OK(c, pageData)
	}
}

func parseCertsInfo(secret corev1.Secret) map[string]certificate.Certificate {
	if secret.Type != corev1.SecretTypeTLS {
		return nil
	}
	ret := map[string]certificate.Certificate{}
	for k, v := range secret.Data {
		// tls.crt ca.crt
		if k != corev1.TLSCertKey && k != corev1.ServiceAccountRootCAKey {
			continue
		}
		cert, err := certificate.ParseCertInfo(v)
		if err != nil {
			continue
		}
		ret[k] = *cert
	}
	return ret
}
