// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argo

import (
	"fmt"

	"kubegems.io/kubegems/pkg/apis/gems"
)

type Options struct {
	Addr     string `json:"addr" description:"argocd host"`
	Token    string `json:"token" description:"argocd token,if empty generate from username password"`
	Username string `json:"username" description:"argocd username"`
	Password string `json:"password" description:"argocd password"`
}

func NewDefaultArgoOptions() *Options {
	return &Options{
		Addr: fmt.Sprintf("http://kubegems-argo-cd-server.%s", gems.NamespaceSystem),
		// 保持为空，则使用 kube port forward
		// https://argoproj.github.io/argo-cd/operator-manual/security/#authentication
		// 使用的是第 1 种方式，先 user/password 方式,admin登录web页面,从请求中拿到 admin token
		Token:    "",
		Username: "admin",
	}
}
