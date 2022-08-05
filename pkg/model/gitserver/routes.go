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
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"kubegems.io/kubegems/pkg/log"
)

const mimeGitLFSJSON = "application/vnd.git-lfs+json"

func LFSBatchMatcher(r *http.Request, m *mux.RouteMatch) bool {
	return strings.Split(r.Header.Get("Accept"), ";")[0] == mimeGitLFSJSON &&
		strings.Split(r.Header.Get("Content-Type"), ";")[0] == mimeGitLFSJSON
}

// nolint: funlen
func (s *Server) routes(lfsenabled bool, githttpbackendenabled bool) http.Handler {
	r := mux.NewRouter()
	repoapi := r.PathPrefix("/{username}/{repository}").Subrouter()
	// admin
	repoapi.HandleFunc("", s.CreateRepository).Methods("POST")
	repoapi.HandleFunc("", s.RemoveRepository).Methods("DELETE")
	repoapi.HandleFunc("/files", s.ListFiles).Methods("GET")

	// .git
	gitrepor := r.PathPrefix("/{username}/{repository}.git").Subrouter()

	// git lfs
	if lfsenabled {
		// https://github.com/git-lfs/git-lfs/blob/main/docs/api/server-discovery.md#server-discovery
		gitlfsr := gitrepor.PathPrefix("/info/lfs").Subrouter()
		gitlfsr.HandleFunc("/objects/batch", s.LFSBatch).Methods("POST").MatcherFunc(LFSBatchMatcher)
		// https://github.com/git-lfs/git-lfs/blob/main/docs/api/basic-transfers.md#basic-transfer-api
		gitlfsr.HandleFunc("/objects", s.LFSUpload).Methods("POST")
		gitlfsr.HandleFunc("/objects/{oid}", s.LFSDownload).Methods("GET")
		gitlfsr.HandleFunc("/objects/{oid}", s.LFSUpdate).Methods("PUT")
		gitlfsr.HandleFunc("/objects/{oid}", s.LFSDelete).Methods("DELETE")
		// https://github.com/git-lfs/git-lfs/blob/main/docs/api/basic-transfers.md#verification
		gitlfsr.HandleFunc("/verify", s.LFSVerify).Methods("POST").MatcherFunc(LFSBatchMatcher)
	}

	// git http
	if githttpbackendenabled {
		// git http backend
		// (HEAD|info/refs|objects/(info/[^/]+|[0-9a-f]{2}/[0-9a-f]{38}|pack/pack-[0-9a-f]{40}\.(pack|idx))|git-(upload|receive)-pack)
		paths := []string{
			"/HEAD",
			"/info/refs",
			"/objects/info/{-:[^/]+}",
			"/objects/{-:[0-9a-f]{2}/[0-9a-f]{38}}",
			"/objects/{-:[0-9a-f]{2}/[0-9a-f]{62}}",
			"/objects/pack/pack-{-:[0-9a-f]{40}}.pack",
			"/objects/pack/pack-{-:[0-9a-f]{64}}.pack",
			"/objects/pack/pack-{-:[0-9a-f]{40}}.idx",
			"/objects/pack/pack-{-:[0-9a-f]{64}}.idx",
			"/git-upload-pack",
			"/git-receive-pack",
		}
		for _, path := range paths {
			gitrepor.HandleFunc(path, s.GitHTTPBackend).Methods("GET", "POST")
		}
	} else {
		// smart http
		gitrepor.HandleFunc("/info/refs", s.GetInfoRefsWithService).Methods("GET").Queries("service", "{servicename:.*}")
		gitrepor.HandleFunc("/git-upload-pack", s.UploadPack).Methods("POST")
		gitrepor.HandleFunc("/git-receive-pack", s.ReceivePack).Methods("POST")
		// dumb http
		gitrepor.HandleFunc("/HEAD", s.GetHead).Methods("GET")
		gitrepor.HandleFunc("/info/refs", s.GetInfoRefs).Methods("GET")
		gitrepor.HandleFunc("/objects/info/alternates", s.GetAlternative).Methods("GET")
		gitrepor.HandleFunc("/objects/info/http-alternates", s.GetHTTPAlternative).Methods("GET")
		gitrepor.HandleFunc("/objects/info/packs", s.GetInfoPacks).Methods("GET")
		gitrepor.HandleFunc("/objects/{hash-dir:[0-9a-f]{2}}/{hash:[0-9a-f]{38}}", s.GetLooseObject).Methods("GET")
		gitrepor.HandleFunc("/objects/{hash-dir:[0-9a-f]{2}}/{hash:[0-9a-f]{62}}", s.GetLooseObject).Methods("GET")
		gitrepor.HandleFunc("/objects/pack/pack-{hash:[0-9a-f]{40}}.pack", s.GetPackFile).Methods("GET")
		gitrepor.HandleFunc("/objects/pack/pack-{hash:[0-9a-f]{64}}.pack", s.GetPackFile).Methods("GET")
		gitrepor.HandleFunc("/objects/pack/pack-{hash:[0-9a-f]{40}}.idx", s.GetIdxFile).Methods("GET")
		gitrepor.HandleFunc("/objects/pack/pack-{hash:[0-9a-f]{64}}.idx", s.GetIdxFile).Methods("GET")
	}
	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(LoggingMiddleware)
	return r
}

func LoggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Infof("%s %s %s", r.Method, r.URL, time.Since(start))
	})
}
