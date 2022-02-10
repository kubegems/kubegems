package application

import "testing"

func TestPathRef_FullName(t *testing.T) {
	type fields struct {
		Tenant  string
		Project string
		Env     string
		Name    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			fields: fields{
				Tenant:  "t",
				Project: "p",
				Env:     "e",
				Name:    "longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong",
			},
			want: "5a638767-t-p-e-longlonglonglonglonglonglonglong",
		},
		{
			fields: fields{
				Tenant:  "tdasgoucgdsaiugyfiur3",
				Project: "fewklgviwhigver.-kfdkjsgv",
				Env:     "e321gdiuqwkfwelkf",
				Name:    "longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong",
			},
			want: "ef492a8e-tda-few-e32-longlonglonglonglonglonglonglong",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PathRef{
				Tenant:  tt.fields.Tenant,
				Project: tt.fields.Project,
				Env:     tt.fields.Env,
				Name:    tt.fields.Name,
			}
			if got := p.FullName(); got != tt.want {
				t.Errorf("PathRef.FullName() = %v, want %v", got, tt.want)
			}
		})
	}
}
