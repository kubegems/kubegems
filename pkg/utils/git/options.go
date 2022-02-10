package git

import (
	"github.com/kubegems/gems/pkg/utils"
	"github.com/spf13/pflag"
)

type Commiter struct {
	Name  string
	Email string
}

type Options struct {
	Host      string    `yaml:"host"`
	Token     string    `yaml:"token"` // 暂时不支持该方式配置,argo 使用到了该处配置，argo 不支持 token 方式
	Username  string    `yaml:"username"`
	Password  string    `yaml:"password"`
	Committer *Commiter `yaml:"committer"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Host:     "http://gems-gitea:3000",
		Username: "root",
		Password: "",
		Committer: &Commiter{
			Name:  "service",
			Email: "service@example.com",
		},
	}
}

func (o *Options) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.Host, utils.JoinFlagName(prefix, "host"), o.Host, "git host")
	fs.StringVar(&o.Username, utils.JoinFlagName(prefix, "username"), o.Username, "git username")
	fs.StringVar(&o.Password, utils.JoinFlagName(prefix, "password"), o.Password, "git password")
}
