package argo

import (
	gemlabels "github.com/kubegems/gems/pkg/labels"
	"github.com/kubegems/gems/pkg/utils"
	"github.com/spf13/pflag"
)

type Options struct {
	Addr      string `yaml:"addr"`
	Token     string `yaml:"token"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Namespace string `yaml:"namespace"`
}

func (o *Options) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.Addr, utils.JoinFlagName(prefix, "addr"), o.Addr, "argocd host")
	fs.StringVar(&o.Token, utils.JoinFlagName(prefix, "token"), o.Token, "argocd token")
	fs.StringVar(&o.Username, utils.JoinFlagName(prefix, "username"), o.Username, "argocd username")
	fs.StringVar(&o.Password, utils.JoinFlagName(prefix, "password"), o.Password, "argocd password")
	fs.StringVar(&o.Namespace, utils.JoinFlagName(prefix, "namespace"), o.Namespace, "argocd namespace")
}

func NewDefaultArgoOptions() *Options {
	return &Options{
		Addr: "http://argocd-server.gemcloud-workflow-system",
		// 保持为空，则使用 kube port forward
		// https://argoproj.github.io/argo-cd/operator-manual/security/#authentication
		// 使用的是第 1 种方式，先 user/password 方式,admin登录web页面,从请求中拿到 admin token
		Token:     "",
		Namespace: gemlabels.NamespaceWorkflow,
	}
}
