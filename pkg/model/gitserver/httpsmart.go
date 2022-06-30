package gitserver

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const GitUploackPack = "application/x-git-upload-pack-advertisement"

func (s *Server) GetInfoRefsWithService(w http.ResponseWriter, r *http.Request) {
	servicename := r.FormValue("service")
	repopath := s.RepositoryPath(r)
	ctx := r.Context()

	envs := []string{}
	for k, v := range r.Header {
		k = strings.Map(UpperCaseAndUnderscore, k)
		envs = append(envs, "HTTP_"+k+"="+strings.Join(v, ", "))
	}
	if protocol := r.Header.Get("Git-Protocol"); protocol != "" {
		envs = append(envs, "GIT_PROTOCOL="+protocol)
	}

	cmd := exec.CommandContext(ctx, "git", strings.TrimPrefix(servicename, "git-"), "--stateless-rpc", "--advertise-refs", ".")
	cmd.Dir = filepath.Join(s.GitBase, repopath)
	cmd.Env = envs

	errbuf, outbuf := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = r.Body, outbuf, io.MultiWriter(errbuf, os.Stderr)
	if err := cmd.Run(); err != nil {
		InternalServerError(w, err.Error())
		return
	}
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", servicename))
	SetHeaderNoCache(w)
	w.WriteHeader(http.StatusOK)
	w.Write(packetWrite("# service=" + servicename + "\n"))
	w.Write([]byte("0000"))
	w.Write(outbuf.Bytes())
}

// nolint: gomnd
func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)
	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}
	return []byte(s + str)
}

func (s *Server) UploadPack(w http.ResponseWriter, r *http.Request) {
	wd := filepath.Join(s.GitBase, s.RepositoryPath(r))
	GitServiceCall(w, r, wd, "upload-pack", "--stateless-rpc", ".")
}

func (s *Server) ReceivePack(w http.ResponseWriter, r *http.Request) {
	wd := filepath.Join(s.GitBase, s.RepositoryPath(r))
	GitServiceCall(w, r, wd, "receive-pack", "--stateless-rpc", ".")
}

func GitServiceCall(w http.ResponseWriter, r *http.Request, repopath, servicename string, args ...string) {
	if r.Header.Get("Content-Encoding") == "gzip" {
		reqBody, err := gzip.NewReader(r.Body)
		if err != nil {
			InternalServerError(w, err.Error())
			return
		}
		r.Body = reqBody
	}

	arg0 := strings.TrimPrefix(servicename, "git-")
	envs := []string{
		"SSH_ORIGINAL_COMMAND=" + arg0,
	}
	envs = append(envs, os.Environ()...)
	if protocol := r.Header.Get("Git-Protocol"); protocol != "" {
		envs = append(envs, "GIT_PROTOCOL="+protocol)
	}

	args = append([]string{arg0}, args...)
	cmd := exec.CommandContext(r.Context(), "git", args...)
	cmd.Env = envs
	cmd.Dir = repopath
	cmd.Stdin, cmd.Stdout, cmd.Stderr = r.Body, w, os.Stderr
	if err := cmd.Run(); err != nil {
		InternalServerError(w, err.Error())
		return
	}
}

func UpperCaseAndUnderscore(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - ('a' - 'A')
	case r == '-':
		return '_'
	case r == '=':
		// Maybe not part of the CGI 'spec' but would mess up
		// the environment in any case, as Go represents the
		// environment as a slice of "key=value" strings.
		return '_'
	}
	// TODO: other transformations in spec or practice?
	return r
}
