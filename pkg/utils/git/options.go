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
