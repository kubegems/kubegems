package application

import "testing"

func TestTaskNameOf(t *testing.T) {
	type args struct {
		ref      PathRef
		taskname string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{
				ref:      PathRef{Tenant: "ten", Project: "proj", Env: "env", Name: "name"},
				taskname: "deploy",
			},
			want: "ten/proj/env/name/deploy",
		},
		{
			args: args{
				ref: PathRef{Tenant: "ten", Project: "proj", Env: "env"},
			},
			want: "ten/proj/env/",
		},
		{
			args: args{
				ref: PathRef{Tenant: "ten", Project: "proj"},
			},
			want: "ten/proj/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TaskNameOf(tt.args.ref, tt.args.taskname); got != tt.want {
				t.Errorf("TaskNameOf() = %v, want %v", got, tt.want)
			}
		})
	}
}
