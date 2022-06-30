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
