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

package common

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

func EncodeToX509Pair(in tls.Certificate) ([]byte, []byte) {
	cert := pem.EncodeToMemory(&pem.Block{Type: cert.CertificateBlockType, Bytes: in.Certificate[0]})
	key, _ := keyutil.MarshalPrivateKeyToPEM(in.PrivateKey)
	return cert, key
}

const (
	DurationYear = time.Hour * 24 * 365
)

type CertOptions struct {
	CommonName string
	Hosts      []string
	ExpireAt   *time.Time
}

// nolint: gomnd,funlen
func SignCertificate(caPEMBlock, certPEMBlock, keyPEMBlock []byte, options CertOptions) ([]byte, []byte, error) {
	tlscert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return nil, nil, err
	}
	intermediateCADERBytes := tlscert.Certificate[0]
	parent, err := x509.ParseCertificate(intermediateCADERBytes)
	if err != nil {
		return nil, nil, err
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	validFrom := time.Now().Add(-time.Hour)
	validTo := validFrom.Add(DurationYear)
	if options.ExpireAt != nil && !options.ExpireAt.IsZero() {
		validTo = *options.ExpireAt
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: options.CommonName},
		NotBefore:    validFrom,
		NotAfter:     validTo,
		// PublicKey: ,
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth | x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		Extensions:            []pkix.Extension{},
	}
	for _, val := range append(options.Hosts, options.CommonName) {
		if val == "" {
			continue
		}
		if ip := net.ParseIP(val); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, val)
		}
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parent, &priv.PublicKey, tlscert.PrivateKey)
	if err != nil {
		return nil, nil, err
	}
	// Generate cert, followed by ca
	certBuffer := bytes.Buffer{}
	if err := pem.Encode(&certBuffer, &pem.Block{Type: cert.CertificateBlockType, Bytes: derBytes}); err != nil {
		return nil, nil, err
	}
	if err := pem.Encode(&certBuffer, &pem.Block{Type: cert.CertificateBlockType, Bytes: intermediateCADERBytes}); err != nil {
		return nil, nil, err
	}
	// Generate key
	keyBuffer := bytes.Buffer{}
	if err := pem.Encode(&keyBuffer, &pem.Block{Type: keyutil.RSAPrivateKeyBlockType, Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return nil, nil, err
	}
	return certBuffer.Bytes(), keyBuffer.Bytes(), err
}
