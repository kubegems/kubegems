package clusterinfo

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	APIServerURL    = "https://kubernetes.default:443"
	APIServerCertCN = "apiserver"
)

func GetServerCertExpiredTime(serverURL string, cn string) (*time.Time, error) {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	u, err := url.Parse(APIServerURL)
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
		if strings.Contains(cert.Subject.CommonName, cn) {
			return &cert.NotAfter, nil
		}
		invalidCNs = append(invalidCNs, cert.Subject.CommonName)
	}

	return nil, fmt.Errorf("cert CN not contains apiserver: %s", strings.Join(invalidCNs, ","))
}
