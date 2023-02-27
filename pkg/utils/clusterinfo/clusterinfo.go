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

package clusterinfo

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	APIServerURL       = "https://kubernetes.default:443"
	K8sAPIServerCertCN = "apiserver"
	K3sAPIServerCertCN = "k3s"
)

func GetServerCertExpiredTime(serverURL string) (*time.Time, error) {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	conn, err := tls.Dial("tcp", u.Host, conf)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	invalidCNs := []string{}
	for _, cert := range conn.ConnectionState().PeerCertificates {
		if strings.Contains(cert.Subject.CommonName, K8sAPIServerCertCN) ||
			strings.Contains(cert.Subject.CommonName, K3sAPIServerCertCN) {
			return &cert.NotAfter, nil
		}
		invalidCNs = append(invalidCNs, cert.Subject.CommonName)
	}

	return nil, fmt.Errorf("cert CN not contains apiserver: %s", strings.Join(invalidCNs, ","))
}
