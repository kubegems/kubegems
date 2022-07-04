package git

import (
	"fmt"

	"kubegems.io/kubegems/pkg/apis/gems"
)

var DefaultCommiter = &Commiter{
	Name:  "service",
	Email: "service@kubgems.io",
}

type Commiter struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type Options struct {
	Addr     string `json:"addr" description:"git addr"`
	Username string `json:"username" description:"git username"`
	Password string `json:"password" description:"git password"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Addr:     fmt.Sprintf("http://kubegems-gitea-http.%s:3000", gems.NamespaceSystem),
		Username: "root",
		Password: "",
	}
}
