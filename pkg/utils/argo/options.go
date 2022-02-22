package argo

type Options struct {
	Addr     string `json:"addr" description:"argocd host"`
	Token    string `json:"token" description:"argocd token,if empty generate from username password"`
	Username string `json:"username" description:"argocd username"`
	Password string `json:"password" description:"argocd password"`
}

func NewDefaultArgoOptions() *Options {
	return &Options{
		Addr: "http://argocd-server.gemcloud-workflow-system",
		// 保持为空，则使用 kube port forward
		// https://argoproj.github.io/argo-cd/operator-manual/security/#authentication
		// 使用的是第 1 种方式，先 user/password 方式,admin登录web页面,从请求中拿到 admin token
		Token: "",
	}
}
