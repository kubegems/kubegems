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
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gorilla/mux"
)

// nolint: gomnd
func (s *Server) CreateRepository(w http.ResponseWriter, r *http.Request) {
	initbarefunc := func(barepath string) error {
		if err := os.MkdirAll(barepath, 0o755); err != nil {
			return err
		}
		if _, err := CallGit(barepath, "init", "--initial-branch=main", "--bare", "--shared", "."); err != nil {
			return err
		}
		if _, err := CallGit(barepath, "config", "http.receivepack", "true"); err != nil {
			return err
		}
		postUpdateHook := `exec git update-server-info`
		if err := os.WriteFile(filepath.Join(barepath, "hooks", "post-update"), []byte(postUpdateHook), 0o755); err != nil {
			return err
		}
		return nil
	}
	if err := initbarefunc(filepath.Join(s.GitBase, s.RepositoryPath(r))); err != nil {
		BadRequest(w, err.Error())
	} else {
		RawResponse(w, http.StatusCreated, nil, nil)
	}
}

func (s *Server) RemoveRepository(w http.ResponseWriter, r *http.Request) {
	repopath := s.RepositoryPath(r)
	if err := os.RemoveAll(filepath.Join(s.GitBase, repopath)); err != nil {
		BadRequest(w, err.Error())
	} else {
		OK(w, "")
	}
}

func (s *Server) RepositoryPath(r *http.Request) string {
	vars := mux.Vars(r)
	username, repository := vars["username"], vars["repository"]
	subpath := filepath.Join(username, repository+".git")
	return subpath
}

func (s *Server) OnRepository(w http.ResponseWriter, r *http.Request, fun func(repofs fs.FS)) {
	repoapth := s.RepositoryPath(r)
	subfs := os.DirFS(filepath.Join(s.GitBase, repoapth))
	fun(subfs)
}

func CallGit(wd string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = wd
	return cmd.CombinedOutput()
}
