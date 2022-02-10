package workflow

import (
	"context"
	"testing"
)

func TestIdentityKeyOfFunction(t *testing.T) {
	type args struct {
		fun interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "named func",
			args: args{
				fun: Namedtestfunc,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IdentityKeyOfFunction(tt.args.fun); got != tt.want {
				t.Errorf("IdentityKeyOfFunction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Namedtestfunc(ctx context.Context) error {
	return nil
}
