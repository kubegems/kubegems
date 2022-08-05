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

package gitserver

import (
	"log"
	"net/http"
	"net/http/cgi"
	"os/exec"
)

func (s *Server) GitHTTPBackend(w http.ResponseWriter, r *http.Request) {
	gitpath, err := exec.LookPath("git")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h := cgi.Handler{
		Path: gitpath,
		Args: []string{"http-backend"},
		Dir:  s.GitBase,
		Env: []string{
			"GIT_PROJECT_ROOT=" + s.GitBase,
			"GIT_HTTP_EXPORT_ALL=on",
		},
		Logger: log.Default(),
	}
	h.ServeHTTP(w, r)
}
