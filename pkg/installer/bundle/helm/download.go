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

package helm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

const (
	DefaultFileMode      os.FileMode = 0o644
	DefaultDirectoryMode os.FileMode = 0o755
)

// LocateChart looks for a chart directory in known places, and returns either the full path or an error.
func LocateChart(ctx context.Context, repoURL, name, version string, cachedir string) (string, error) {
	name, version = strings.TrimSpace(name), strings.TrimSpace(version)
	// check local directory
	if _, err := os.Stat(name); err == nil {
		return filepath.Abs(name)
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
		return name, fmt.Errorf("path %s not found", name)
	}
	if repoURL == "" {
		return name, fmt.Errorf("repo for %s not set", name)
	}
	// nolint: gomnd
	if err := os.MkdirAll(cachedir, 0o755); err != nil {
		return "", err
	}
	getters := []getter.Provider{{Schemes: []string{"http", "https"}, New: getter.NewHTTPGetter}}
	chartdownloadurl, err := repo.FindChartInRepoURL(repoURL, name, version, "", "", "", getters)
	if err != nil {
		return "", err
	}
	data, err := HTTPGet(ctx, chartdownloadurl)
	if err != nil {
		return "", err
	}
	defer data.Close()

	destfile := filepath.Join(cachedir, path.Base(chartdownloadurl))
	// nolint: gomnd
	if err := AtomicWriteFile(destfile, data, 0o644); err != nil {
		return destfile, err
	}
	return filepath.Abs(destfile)
}

func HTTPGet(ctx context.Context, href string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, href, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s : %s", href, resp.Status)
	}
	return resp.Body, nil
}

func AtomicWriteFile(filename string, reader io.Reader, mode os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Split(filename))
	if err != nil {
		return err
	}
	tempName := tempFile.Name()
	if _, err := io.Copy(tempFile, reader); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempName, mode); err != nil {
		return fmt.Errorf("cannot chmod %s to %s: %w", tempName, mode, err)
	}
	if err := RenameFile(tempName, filename); err != nil {
		return fmt.Errorf("cannot rename %s to %s: %w", tempName, filename, err)
	}
	return nil
}

func RenameFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	terr, ok := err.(*os.LinkError)
	if !ok {
		return err
	}
	if terr.Err != syscall.EXDEV {
		return err
	}
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// nolint: nosnakecase
	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fi.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func ExpandChart(chartfile string) (string, error) {
	if fi, err := os.Stat(chartfile); err != nil {
		return chartfile, err
	} else if fi.IsDir() {
		return chartfile, nil
	}
	basedir, filename := filepath.Split(chartfile)
	targetdir := filepath.Join(basedir, strings.TrimSuffix(filename, ".tgz"))

	f, err := os.Open(chartfile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := Expand(targetdir, f); err != nil {
		return chartfile, err
	}
	_ = os.RemoveAll(chartfile)
	return targetdir, nil
}

func Expand(dir string, r io.Reader) error {
	files, err := loader.LoadArchiveFiles(r)
	if err != nil {
		return err
	}
	for _, file := range files {
		outpath := filepath.Join(dir, file.Name)
		if err := os.MkdirAll(filepath.Dir(outpath), DefaultDirectoryMode); err != nil {
			return err
		}
		if err := os.WriteFile(outpath, file.Data, DefaultFileMode); err != nil {
			return err
		}
	}
	return nil
}

type ChartDownloader struct {
	indexCache           sync.Map
	chartsCacheDirectory string
}

func (d *ChartDownloader) GetIndex(ctx context.Context, repoURL string) (*repo.IndexFile, error) {
	if val, ok := d.indexCache.Load(repoURL); ok {
		// nolint: forcetypeassert
		return val.(*repo.IndexFile), nil
	}
	index, err := LoadRemoteIndex(ctx, repoURL)
	if err != nil {
		return nil, err
	}
	d.indexCache.Store(repoURL, index)
	return index, nil
}

func (d *ChartDownloader) download(ctx context.Context, repoURL, chartName, chartVersion string, cachedir string) (string, error) {
	if err := os.MkdirAll(cachedir, DefaultDirectoryMode); err != nil {
		return "", err
	}
	repoIndex, err := d.GetIndex(ctx, repoURL)
	if err != nil {
		return "", err
	}
	cv, err := repoIndex.Get(chartName, chartVersion)
	if err != nil {
		return "", fmt.Errorf("%s-%s not found in repository  %s", chartName, chartVersion, repoURL)
	}
	if len(cv.URLs) == 0 {
		return "", fmt.Errorf("%s-%s has no downloadable URLs", chartName, chartVersion)
	}
	chartURL := cv.URLs[0]
	chartdownloadurl, err := repo.ResolveReferenceURL(repoURL, chartURL)
	if err != nil {
		return "", fmt.Errorf("failed to make chart URL absolute: %s", chartURL)
	}
	destfile, err := filepath.Abs(filepath.Join(cachedir, path.Base(chartdownloadurl)))
	if err != nil {
		return "", err
	}

	data, err := HTTPGet(ctx, chartdownloadurl)
	if err != nil {
		return "", err
	}
	defer data.Close()

	if err := AtomicWriteFile(destfile, data, DefaultFileMode); err != nil {
		return "", err
	}
	return destfile, nil
}
