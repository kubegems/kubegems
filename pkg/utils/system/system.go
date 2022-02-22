package system

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

// nolint: lll
type Options struct {
	Listen   string `json:"listen,omitempty" description:"listen address"`
	CAFile   string `json:"caFile,omitempty" description:"ca file path"`
	CertFile string `json:"certFile,omitempty" description:"cert file path"`
	KeyFile  string `json:"keyFile,omitempty" description:"key file path"`
	CertDir  string `json:"certDir,omitempty" description:"cert dir path,if not empty will use certDir and ca.crt tls.crt tls.key instead of caFile certFile and keyFile"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Listen:   ":8080",
		CertDir:  "certs",
		CAFile:   "certs/ca.crt",
		CertFile: "certs/tls.crt",
		KeyFile:  "certs/tls.key",
	}
}

func (o *Options) IsTLSConfigEnabled() bool {
	return (o.CertFile != "" && o.KeyFile != "") || o.CertDir != ""
}

func (o *Options) ToTLSConfig() (*tls.Config, error) {
	cafile, certfile, keyfile := o.CAFile, o.CertFile, o.KeyFile
	if o.CertDir != "" {
		cafile = o.CertDir + "/ca.crt"
		certfile = o.CertDir + "/tls.crt"
		keyfile = o.CertDir + "/tls.key"
	}

	config := &tls.Config{
		ClientCAs: x509.NewCertPool(),
	}

	if cafile != "" {
		capem, err := ioutil.ReadFile(cafile)
		if err != nil {
			return nil, err
		}
		config.ClientCAs.AppendCertsFromPEM(capem)
	}

	if certfile != "" && keyfile != "" {
		certificate, err := tls.LoadX509KeyPair(cafile, keyfile)
		if err != nil {
			return nil, err
		}
		config.Certificates = append(config.Certificates, certificate)
	}
	return config, nil
}
