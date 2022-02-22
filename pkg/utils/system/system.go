package system

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

type Options struct {
	Listen   string `json:"listen,omitempty" description:"listen address"`
	CAFile   string `json:"caFile,omitempty" description:"ca file path"`
	CertFile string `json:"certFile,omitempty" description:"cert file path"`
	KeyFile  string `json:"keyFile,omitempty" description:"key file path"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Listen:   ":8080",
		CAFile:   "",
		CertFile: "",
		KeyFile:  "",
	}
}

func (o *Options) IsTLSConfigEnabled() bool {
	return (o.CertFile != "" && o.KeyFile != "")
}

func (o *Options) ToTLSConfig() (*tls.Config, error) {
	cafile, certfile, keyfile := o.CAFile, o.CertFile, o.KeyFile

	config := &tls.Config{
		ClientCAs: x509.NewCertPool(),
	}

	// ca
	if cafile != "" {
		capem, err := ioutil.ReadFile(cafile)
		if err != nil {
			return nil, err
		}
		config.ClientCAs.AppendCertsFromPEM(capem)
	}

	// cert
	certificate, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, err
	}
	config.Certificates = append(config.Certificates, certificate)

	return config, nil
}
