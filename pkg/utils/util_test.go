package utils

import (
	"testing"
)

func TestRoundTo(t *testing.T) {
	type args struct {
		n        float64
		decimals uint32
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "remain 2",
			args: args{
				n:        3.1415926,
				decimals: 2,
			},
			want: 3.14,
		},
		{
			name: "remain 5",
			args: args{
				n:        3.1415926,
				decimals: 5,
			},
			want: 3.14159,
		},
		{
			name: "round out of range",
			args: args{
				n:        3.14,
				decimals: 3,
			},
			want: 3.14,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoundTo(tt.args.n, tt.args.decimals); got != tt.want {
				t.Errorf("RoundTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinFlagName(t *testing.T) {
	type args struct {
		prefix string
		key    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				prefix: "",
				key:    "Test",
			},
			want: "test",
		},
		{
			name: "normal",
			args: args{
				prefix: "p1",
				key:    "Test",
			},
			want: "p1-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinFlagName(tt.args.prefix, tt.args.key); got != tt.want {
				t.Errorf("JoinFlagName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidPassword(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				input: "Pass123!",
			},
			wantErr: false,
		},
		{
			name: "failed less",
			args: args{
				input: "Pas!123",
			},
			wantErr: true,
		},
		{
			name: "failed upper",
			args: args{
				input: "paas!123",
			},
			wantErr: true,
		},
		{
			name: "failed lower",
			args: args{
				input: "PPPP!123",
			},
			wantErr: true,
		},
		{
			name: "failed number",
			args: args{
				input: "PPPP!sdfsd",
			},
			wantErr: true,
		},
		{
			name: "failed special character",
			args: args{
				input: "PPPP123sdfsd",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidPassword(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("ValidPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
