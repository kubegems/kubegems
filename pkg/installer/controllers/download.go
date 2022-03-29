package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/logr"
)

func Download(ctx context.Context, repo, version, path string) (string, error) {
	logr.FromContextOrDiscard(ctx).
		WithValues("repo", repo, "version", version, "path", path).
		Info("downloading...")

	if repo == "" || path == "" {
		return filepath.Join(repo, path), nil
	}
	// download
	u, err := url.Parse(repo)
	if err != nil {
		return "", err
	}
	// file://
	if u.Scheme == "file" || u.Scheme == "" {
		abs, err := filepath.Abs(u.Host)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, u.Path, path), nil
	}
	// git
	if strings.HasSuffix(u.Path, ".git") {
		cachedir, err := NewCacheDir(repo, version)
		if err != nil {
			return "", err
		}
		if err := DownloadGit(ctx, repo, version, cachedir); err != nil {
			return "", err
		}
		return filepath.Join(cachedir, path), nil
	}
	// unsupported
	return "", fmt.Errorf("unsupported download url: %s", repo)
}

func NewCacheDir(repo, version string) (string, error) {
	u, err := url.Parse(repo)
	if err != nil {
		return "", err
	}
	return filepath.Join(os.TempDir(), "kubegems-cache", u.Host, u.Path+"@"+version), nil
}

func DownloadGit(ctx context.Context, cloneurl string, rev string, dir string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("cloneurl", cloneurl, "rev", rev, "dir", dir)

	log.Info("cloning...")
	repository, err := git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
		URL: cloneurl,
		// using git clone --depth 1 when this issues is fixed
		// https://github.com/go-git/go-git/pull/447
		// Depth:           1,
		Tags:            git.AllTags,
		InsecureSkipTLS: true,
	})
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return err
		}
		log.Info("already exists, updating...")
		repository, err = git.PlainOpen(dir)
		if err != nil {
			return err
		}
		// git remote update
		remotes, err := repository.Remotes()
		if err != nil {
			return err
		}
		for _, remote := range remotes {
			remote.FetchContext(ctx, &git.FetchOptions{InsecureSkipTLS: true})
		}
	}
	wt, err := repository.Worktree()
	if err != nil {
		return err
	}

	hash, err := repository.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return err
	}
	// git reset --hard <hash>
	if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: *hash}); err != nil {
		return err
	}
	return nil
}
