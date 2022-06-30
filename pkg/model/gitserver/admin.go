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
		if _, err := CallGit(barepath, "init", "--bare", "--shared", "."); err != nil {
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
