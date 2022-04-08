package controllers

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
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
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
)

const (
	defaultDirMode  = 0o755
	defaultFileMode = 0o644
)

// we cache "plugin" in a directory with name "{name}-{version}" under cache directory
func DownloadPlugin(ctx context.Context, plugin *Plugin, cachedir string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("plugin", plugin.Name)

	pluginCacheDir := ""
	if plugin.Version == "" {
		pluginCacheDir = filepath.Join(cachedir, plugin.Name)
	} else {
		pluginCacheDir = filepath.Join(cachedir, plugin.Name+"-"+plugin.Version)
	}

	// cache hint?
	if entries, err := os.ReadDir(pluginCacheDir); err == nil && len(entries) > 0 {
		log.Info("already download,use cache")
		plugin.Path = pluginCacheDir
		return nil
	}

	if plugin.Repo == "" {
		return fmt.Errorf("plugin %s no repo set and not found in cache dir %s", plugin.Name, pluginCacheDir)
	}

	log.Info("downloading...")
	dlrepo := DownloadRepo{
		URI:     plugin.Repo,
		SubPath: plugin.Path,
		Version: plugin.Version,
		Name:    plugin.Name,
	}
	if err := Download(ctx, dlrepo, pluginCacheDir); err != nil {
		log.Error(err, "on download")
		return err
	}
	log.Info("download finished")
	plugin.Path = pluginCacheDir
	return nil
}

type Downloader struct {
	CacheDir string
}

// cases
// 1. URI: charts.example.com/repository
// 1. URI: files.example.com/blob/filename.tgz
// 1. URI: git.example.com/foo/bar.git														Subpath: deploy/manifests
// 1. URI: https://github.com/rancher/local-path-provisioner/archive/refs/tags/v0.0.22.zip	Subpath: deploy/manifests
// 1. URI: https://github.com/rancher/local-path-provisioner/archive/refs/heads/master.zip 	Subpath:

type DownloadRepo struct {
	URI     string
	SubPath string
	Version string
	Name    string
}

func Download(ctx context.Context, repo DownloadRepo, intodir string) error {
	u, err := url.ParseRequestURI(repo.URI)
	if err != nil {
		return err
	}

	if repo.SubPath != "" && !strings.HasSuffix(repo.SubPath, "/") {
		repo.SubPath += "/"
	}

	// is local path ?
	if u.Scheme == "file" || u.Scheme == "" {
		return DownloadFile(ctx, repo.URI, repo.SubPath, intodir)
	}
	// is git ?
	if strings.HasSuffix(u.Path, ".git") {
		return DownloadGit(ctx, repo.URI, repo.Version, repo.SubPath, intodir)
	}
	// is zip ?
	if strings.HasSuffix(u.Path, ".zip") {
		return DownloadZip(ctx, repo.URI, repo.SubPath, intodir)
	}
	// is tar.gz ?
	if strings.HasSuffix(u.Path, ".tar.gz") || strings.HasSuffix(u.Path, ".tgz") {
		return DownloadTgz(ctx, repo.URI, repo.SubPath, intodir)
	}
	// is helm repo?
	if MayHelm(ctx, repo.URI) {
		return DownloadHelmChart(ctx, repo.URI, repo.Name, repo.Version, intodir)
	}
	return fmt.Errorf("unsupported scheme: %s", u.Scheme)
}

func DownloadZip(ctx context.Context, uri, subpath, into string) error {
	resp, err := http.Get(uri)
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
	resp, err := http.Get(uri)
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

func DownloadHelmChartOriginal(ctx context.Context, repo, name, version string) (string, *chart.Chart, error) {
	chartPathOptions := action.ChartPathOptions{RepoURL: repo, Version: version}
	settings := cli.New()
	chartPath, err := chartPathOptions.LocateChart(name, settings)
	if err != nil {
		return "", nil, err
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", nil, err
	}
	// dependencies update
	if err := action.CheckDependencies(chart, chart.Metadata.Dependencies); err != nil {
		man := &downloader.Manager{
			Out:              io.Discard,
			ChartPath:        chartPath,
			Keyring:          chartPathOptions.Keyring,
			SkipUpdate:       false,
			Getters:          getter.All(settings),
			RepositoryConfig: settings.RepositoryConfig,
			RepositoryCache:  settings.RepositoryCache,
			Debug:            settings.Debug,
		}
		if err := man.Update(); err != nil {
			return "", nil, err
		}
		chart, err = loader.Load(chartPath)
		if err != nil {
			return "", nil, err
		}
	}
	return chartPath, chart, err
}

func DownloadHelmChart(ctx context.Context, repo, name, version, intodir string) error {
	chartPath, chart, err := DownloadHelmChartOriginal(ctx, repo, name, version)
	if err != nil {
		return err
	}
	// untgz chartPath into intodir
	f, err := os.Open(chartPath)
	if err != nil {
		return err
	}
	return UnTarGz(f, chart.Name(), intodir)
}

func MayHelm(ctx context.Context, uri string) bool {
	indexfile := uri + "/index.yaml"
	resp, err := http.Get(indexfile)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	// todo: more check
	return true
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

		if dir := filepath.Dir(filename); dir != "" {
			if err := os.MkdirAll(dir, defaultDirMode); err != nil {
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
