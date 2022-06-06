package certificate

import (
	"testing"
)

func TestParseCertInfo(t *testing.T) {
	type args struct {
		certPEM []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *Certificate
		wantErr bool
	}{
		{
			name: "",
			args: args{
				certPEM: []byte(`-----BEGIN CERTIFICATE-----
MIIDGzCCAgOgAwIBAgIRAIGHcOrNpwTQH4UOGcGQnfIwDQYJKoZIhvcNAQELBQAw
ADAeFw0yMjA2MDYwMjM3MDlaFw0yMjA5MDQwMjM3MDlaMAAwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQDsybjanc9GZ7FwxLKClc+8bZowE0jmkTT10Y60
Cq4WlEfkZB16iLbFUmCoHvxkq3FEZcd98H+voLEczuJ1DrLx21d7xS+0AxarZGbU
Y5Oqe2J3+qk3QlHbGQEFW5cZZFSBjGKf/TQMUGIvn13ZJ0/+Ha1v+NjjTJ3dqkfb
x+xefh/ygo3159BapgGC8ohToG+oNe2zwrnudGFwqPKFv9IKjEYLUEf+e1DIi0Wj
4gm4edi3KDeE1jnkWzq8mtcorZkexH1M3wfKie881DIaSQW5K39tyYtuM/RSieGG
wvBh3j5RDBMIOmWC3MwIXkLZOqF3UfoGr2w31I5GK0yTdAQhAgMBAAGjgY8wgYww
DgYDVR0PAQH/BAQDAgWgMAwGA1UdEwEB/wQCMAAwbAYDVR0RAQH/BGIwYIIna3Vi
ZWdlbXMtbG9jYWwtYWdlbnQua3ViZWdlbXMtbG9jYWwuc3ZjgjVrdWJlZ2Vtcy1s
b2NhbC1hZ2VudC5rdWJlZ2Vtcy1sb2NhbC5zdmMuY2x1c3Rlci5sb2NhbDANBgkq
hkiG9w0BAQsFAAOCAQEALWLVgGuxxCCKbr0Xqxa/cbModfB1A9EENTv5cP3KJxJt
b71twwOZOKECgkMcbSj/CQFeGbL6JpwhLJ4iSNfvlDfCK8FB02uCFtk2mQ7YRYWz
funLTgovIRnccMZNdvxaVPAZ9feOUr2bUBX75s6myfSsIi3h6sqwAsp+UOOTjuND
PWV7OitYDpscScdBK0z1JUrpC4htOytXhs0hLCvN5rAfrMB5WR61/7ZTDn8rxytr
RaDPwmn9di6rji+Q4BVN1qKhh2AOgGiEK/qVNywAg7s4OOLVN4Se2gwuxlYOlNEn
3eoySEwhETisDmd+y6qh8PM/CXHrCFfFIYKz5SDVFw==
-----END CERTIFICATE-----`),
			},
			want:    &Certificate{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCertInfo(tt.args.certPEM)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCertInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			_ = got
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("ParseCertInfo() = %v, want %v", got, tt.want)
			// }
		})
	}
}
