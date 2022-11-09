package options

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

type TLS struct {
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
	CAFile   string `json:"caFile,omitempty"`
}

func NewDefaultTLS() *TLS {
	return &TLS{
		CAFile:   "certs/ca.crt",
		CertFile: "certs/tls.crt",
		KeyFile:  "certs/tls.key",
	}
}

func (o TLS) ToTLSConfig() (*tls.Config, error) {
	// certs
	certificate, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}
	// ca
	if o.CAFile != "" {
		capem, err := os.ReadFile(o.CAFile)
		if err != nil {
			return nil, err
		}
		cas := x509.NewCertPool()
		cas.AppendCertsFromPEM(capem)
		config.ClientCAs = cas
		config.RootCAs = cas
	}
	return config, nil
}
