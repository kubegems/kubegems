package certificate

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// nolint: tagliatelle
type Certificate struct {
	Subject            Name      `json:"subject,omitempty"`
	Issuer             Name      `json:"issuer,omitempty"`
	SerialNumber       string    `json:"serial_number,omitempty"`
	SANs               []string  `json:"sans,omitempty"`
	NotBefore          time.Time `json:"not_before"`
	NotAfter           time.Time `json:"not_after"`
	SignatureAlgorithm string    `json:"sigalg"`
	AKI                string    `json:"authority_key_id"`
	SKI                string    `json:"subject_key_id"`
	RawPEM             string    `json:"pem"`
}

// nolint: tagliatelle
type Name struct {
	CommonName         string                       `json:"common_name,omitempty"`
	SerialNumber       string                       `json:"serial_number,omitempty"`
	Country            string                       `json:"country,omitempty"`
	Organization       string                       `json:"organization,omitempty"`
	OrganizationalUnit string                       `json:"organizational_unit,omitempty"`
	Locality           string                       `json:"locality,omitempty"`
	Province           string                       `json:"province,omitempty"`
	StreetAddress      string                       `json:"street_address,omitempty"`
	PostalCode         string                       `json:"postal_code,omitempty"`
	Names              []pkix.AttributeTypeAndValue `json:"names,omitempty"`
	ExtraNames         []pkix.AttributeTypeAndValue `json:"extra_names,omitempty"`
}

func ParseCertInfo(certPEM []byte) (*Certificate, error) {
	block, _ := pem.Decode(bytes.TrimSpace(certPEM))
	if block == nil {
		return nil, fmt.Errorf("no certificate PEM data found")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return &Certificate{
		RawPEM:             string(block.Bytes),
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		NotBefore:          cert.NotBefore,
		NotAfter:           cert.NotAfter,
		Subject:            ParseName(cert.Subject),
		Issuer:             ParseName(cert.Issuer),
		SANs: func() []string {
			sans := cert.DNSNames
			for _, ip := range cert.IPAddresses {
				sans = append(sans, ip.String())
			}
			return sans
		}(),
		AKI:          formatKeyID(cert.AuthorityKeyId),
		SKI:          formatKeyID(cert.SubjectKeyId),
		SerialNumber: cert.SerialNumber.String(),
	}, nil
}

func ParseName(name pkix.Name) Name {
	return Name{
		CommonName:         name.CommonName,
		SerialNumber:       name.SerialNumber,
		Country:            strings.Join(name.Country, ","),
		Organization:       strings.Join(name.Organization, ","),
		OrganizationalUnit: strings.Join(name.OrganizationalUnit, ","),
		Locality:           strings.Join(name.Locality, ","),
		Province:           strings.Join(name.Province, ","),
		StreetAddress:      strings.Join(name.StreetAddress, ","),
		PostalCode:         strings.Join(name.PostalCode, ","),
		Names:              name.Names,
		ExtraNames:         name.ExtraNames,
	}
}

func formatKeyID(id []byte) string {
	var s string

	for i, c := range id {
		if i > 0 {
			s += ":"
		}
		s += fmt.Sprintf("%02X", c)
	}

	return s
}
