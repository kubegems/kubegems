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

package bundle

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-logr/logr"
	"kubegems.io/kubegems/pkg/installer/bundle/helm"
)

const (
	defaultDirMode  = 0o755
	defaultFileMode = 0o644
)

type DownloadMeta struct {
	Name    string
	URL     string
	Path    string
	Chart   string
	Version string
}

// we cache "bundle" in a directory with name
// "{repo host}/{name}-{version} or {repo host}/{name}-{version}.tgz" under cache directory
func Download(ctx context.Context, repo, name, version, path, cacheDir string) (string, error) {
	log := logr.FromContextOrDiscard(ctx)
	if name == "" {
		return "", errors.New("empty name")
	}

	if repo == "" {
		return "", fmt.Errorf("no url specified for %s", name)
	}
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache", "kubegems", "bundles")
	}

	basename := name
	if version != "" {
		basename = name + "-" + version
	}
	// from cache
	perRepoCacheDir := PerRepoCacheDir(repo, cacheDir)
	if cachepath := foundInCache(ctx, perRepoCacheDir, basename); cachepath != "" {
		log.Info("found in cache", "path", cachepath)
		return cachepath, nil
	}

	// is file://
	if strings.HasPrefix(repo, "file://") {
		if cachepath := foundInCache(ctx, strings.TrimPrefix(repo, "file://"), basename); cachepath != "" {
			log.Info("found in file protocol", "path", cachepath)
			return cachepath, nil
		}
	}

	cacheIn := filepath.Join(perRepoCacheDir, basename)
	log.Info("downloading...", "cache", cacheIn)

	// is git ?
	if strings.HasSuffix(repo, ".git") {
		return cacheIn, DownloadGit(ctx, repo, version, path, cacheIn)
	}
	// is zip ?
	if strings.HasSuffix(repo, ".zip") {
		return cacheIn, DownloadZip(ctx, repo, path, cacheIn)
	}
	// is tar.gz ?
	if strings.HasSuffix(repo, ".tar.gz") || strings.HasSuffix(repo, ".tgz") {
		return cacheIn, DownloadTgz(ctx, repo, path, cacheIn)
	}
	// is helm ? default helm
	chartpath, _, err := helm.Download(ctx, repo, name, version, filepath.Dir(cacheIn))
	if err != nil {
		return chartpath, err
	}
	return chartpath, err
}

func foundInCache(ctx context.Context, cachedir, basename string) string {
	cacheInFile := filepath.Join(cachedir, basename+".tgz")
	if _, err := os.Stat(cacheInFile); err == nil {
		return cacheInFile
	}
	cacheInDir := filepath.Join(cachedir, basename)
	if entries, err := os.ReadDir(cacheInDir); err == nil && len(entries) >= 0 {
		return cacheInDir
	}
	return ""
}

func PerRepoCacheDir(repo string, basedir string) string {
	repou, err := url.Parse(repo)
	if err != nil {
		return basedir
	}
	if repou.Scheme == "file" {
		return filepath.Join(basedir, filepath.Base(repou.Path))
	} else {
		return filepath.Join(basedir, repou.Hostname())
	}
}

func DownloadZip(ctx context.Context, uri, subpath, into string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	r := bytes.NewReader(raw)
	zipr, err := zip.NewReader(r, r.Size())
	if err != nil {
		return err
	}

	if subpath != "" && !strings.HasSuffix(subpath, "/") {
		subpath += "/"
	}

	for _, file := range zipr.File {
		if !strings.HasPrefix(file.Name, subpath) {
			continue
		}
		{
			filename := strings.TrimPrefix(file.Name, subpath)
			filename = filepath.Join(into, filename)

			if file.FileInfo().IsDir() {
				if err := os.MkdirAll(filename, file.Mode()); err != nil {
					return err
				}
				continue
			}

			dest, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
			if err != nil {
				return err
			}
			defer dest.Close()

			src, err := file.Open()
			if err != nil {
				return err
			}
			defer src.Close()
			_, _ = io.Copy(dest, src)
		}
	}
	return nil
}

func DownloadTgz(ctx context.Context, uri, subpath, into string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return UnTarGz(resp.Body, subpath, into)
}

func DownloadFile(ctx context.Context, src string, subpath, into string) error {
	u, err := url.ParseRequestURI(src)
	if err != nil {
		return err
	}
	if u.Host != "" && u.Host != "localhost" {
		return fmt.Errorf("unsupported host: %s", u.Host)
	}

	basedir := u.Path
	if !strings.HasSuffix(basedir, "/") {
		basedir += "/"
	}

	if err := os.MkdirAll(into, defaultDirMode); err != nil {
		return err
	}

	return filepath.WalkDir(basedir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relpath := strings.TrimPrefix(path, basedir)

		if !strings.HasPrefix(relpath, subpath) {
			return nil
		}

		filename := strings.TrimPrefix(relpath, subpath)
		filename = filepath.Join(into, filename)

		fi, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := os.MkdirAll(filename, fi.Mode().Perm()); err != nil {
				return err
			}
			return nil
		}
		dest, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode().Perm())
		if err != nil {
			return err
		}
		defer dest.Close()

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, _ = io.Copy(dest, f)
		return nil
	})
}

func DownloadGit(ctx context.Context, cloneurl string, rev string, subpath, into string) error {
	repository, err := git.CloneContext(ctx, memory.NewStorage(), nil, &git.CloneOptions{
		URL:          cloneurl,
		Depth:        1,
		SingleBranch: true,
	})
	if err != nil {
		return err
	}

	if rev == "" {
		rev = "HEAD"
	}
	hash, err := repository.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return err
	}

	commit, err := repository.CommitObject(*hash)
	if err != nil {
		return err
	}

	tree, err := repository.TreeObject(commit.TreeHash)
	if err != nil {
		return err
	}

	return tree.Files().ForEach(func(f *object.File) error {
		if !strings.HasPrefix(f.Name, subpath) {
			return nil
		}
		raw, err := f.Contents()
		if err != nil {
			return err
		}

		fmode, err := f.Mode.ToOSFileMode()
		if err != nil {
			fmode = defaultFileMode
		}

		filename := strings.TrimPrefix(f.Name, subpath)
		filename = filepath.Join(into, filename)
		if dir := filepath.Dir(filename); dir != "" {
			if err := os.MkdirAll(dir, defaultDirMode); err != nil {
				return err
			}
		}
		return os.WriteFile(filename, []byte(raw), fmode)
	})
}

func UnTarGz(r io.Reader, subpath, into string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !strings.HasPrefix(hdr.Name, subpath) {
			continue
		}

		filename := strings.TrimPrefix(hdr.Name, subpath)
		filename = filepath.Join(into, filename)

		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(filename, defaultDirMode); err != nil {
				return err
			}
			continue
		} else {
			if err := os.MkdirAll(filepath.Dir(filename), defaultDirMode); err != nil {
				return err
			}
		}

		dest, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer dest.Close()

		_, _ = io.Copy(dest, tr)
	}
	return nil
}

func cachePath(path string, name, version string) string {
	basename := name + "-" + version
	tgzfile := filepath.Join(path, basename+".tgz")
	if _, err := os.Stat(tgzfile); err == nil {
		return tgzfile
	}
	cacheDir := filepath.Join(path, basename)
	if entries, err := os.ReadDir(cacheDir); err == nil && len(entries) >= 0 {
		return cacheDir
	}
	return ""
}

func isNotEmpty(path string) (string, bool) {
	entries, err := os.ReadDir(path)
	return path, (err == nil && len(entries) >= 0)
}

func hasTgz(path string) (string, bool) {
	for _, p := range []string{path + ".tgz", path + ".tar.gz"} {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return path, false
}
