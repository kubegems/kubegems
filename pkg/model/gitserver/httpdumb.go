package gitserver

import (
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
)

const (
	mimeText        = "text/plain; charset=utf-8"
	mimeGitPack     = "application/x-git-packed-objects"
	mimeLooseObject = "application/x-git-loose-object"
	mimeGitPackIdx  = "application/x-git-packed-objects-toc"
)

func (s *Server) GetHead(w http.ResponseWriter, r *http.Request) {
	SetHeaderNoCache(w)
	s.serveFile("HEAD", mimeText, r, w)
}

func (s *Server) GetInfoRefs(w http.ResponseWriter, r *http.Request) {
	SetHeaderNoCache(w)
	s.serveFile("info/refs", mimeText, r, w)
}

func (s *Server) GetAlternative(w http.ResponseWriter, r *http.Request) {
	SetHeaderNoCache(w)
	s.serveFile("objects/info/alternates", mimeText, r, w)
}

func (s *Server) GetHTTPAlternative(w http.ResponseWriter, r *http.Request) {
	SetHeaderNoCache(w)
	s.serveFile("objects/info/alternates", mimeText, r, w)
}

func (s *Server) GetInfoPacks(w http.ResponseWriter, r *http.Request) {
	SetHeaderCacheForever(w)
	s.serveFile("objects/info/packs", mimeGitPack, r, w)
}

func (s *Server) GetLooseObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := filepath.Join("objects", vars["hash-dir"], vars["hash"])
	SetHeaderCacheForever(w)
	s.serveFile(path, mimeLooseObject, r, w)
}

func (s *Server) GetPackFile(w http.ResponseWriter, r *http.Request) {
	path := "objects/pack/pack-" + mux.Vars(r)["hash"] + ".pack"
	SetHeaderCacheForever(w)
	s.serveFile(path, mimeGitPack, r, w)
}

func (s *Server) GetIdxFile(w http.ResponseWriter, r *http.Request) {
	path := "objects/pack/pack-" + mux.Vars(r)["hash"] + ".idx"
	SetHeaderCacheForever(w)
	s.serveFile(path, mimeGitPackIdx, r, w)
}

func (s *Server) serveFile(path string, contentType string, r *http.Request, w http.ResponseWriter) {
	s.OnRepository(w, r, func(repofs fs.FS) {
		file, err := repofs.Open(path)
		if err != nil {
			if err == fs.ErrNotExist {
				NotFound(w)
			} else {
				InternalServerError(w, err.Error())
			}
			return
		}
		defer file.Close()

		if fi, err := file.Stat(); err != nil {
			InternalServerError(w, err.Error())
			return
		} else {
			// nolint: gomnd
			w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
			w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, file)
	})
}
