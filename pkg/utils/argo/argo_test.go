package argo

import "testing"

func TestGetTokenFromUserPassword(t *testing.T) {
	type args struct {
		addr     string
		username string
		password string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTokenFromUserPassword(tt.args.addr, tt.args.username, tt.args.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTokenFromUserPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTokenFromUserPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}
