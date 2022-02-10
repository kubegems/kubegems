package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-logr/logr"
)

const (
	CacheTimeout  = 2 * time.Minute
	DefaultBranch = "_base"
)

type SimpleLocalProvider struct {
	options     *Options
	gitea       *GiteaRemote
	auth        *http.BasicAuth
	caches      sync.Map
	cacheexpire time.Duration
	committer   object.Signature
}

type cacheKey struct {
	Org    string
	Repo   string
	Branch string
}

func NewProvider(options *Options) (*SimpleLocalProvider, error) {
	giteacli, err := gitea.NewClient(options.Host, gitea.SetBasicAuth(options.Username, options.Password))
	if err != nil {
		return nil, err
	}
	return &SimpleLocalProvider{
		options: options,
		gitea: &GiteaRemote{
			Client: giteacli,
		},
		auth: &http.BasicAuth{
			Username: options.Username,
			Password: options.Password,
		},
		committer: object.Signature{
			Name:  options.Committer.Name,
			Email: options.Committer.Email,
		},
		cacheexpire: CacheTimeout,
	}, nil
}

type RepositoryRef struct {
	Org    string `json:"org,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

func (p *SimpleLocalProvider) Options() *Options {
	return p.options
}

func (p *SimpleLocalProvider) GenerateCloneURL(ctx context.Context, ref RepositoryRef) string {
	return fmt.Sprintf("%s/%s/%s.git", p.options.Host, ref.Org, ref.Repo)
}

func (p *SimpleLocalProvider) Get(ctx context.Context, ref RepositoryRef) (*Repository, error) {
	org, repo, branch := ref.Org, ref.Repo, ref.Branch

	if branch == "" {
		branch = DefaultBranch
	}
	target := cacheKey{Org: org, Repo: repo, Branch: branch}

	logger := logr.FromContextOrDiscard(ctx)

	// from cache
	obj, ok := p.caches.Load(target)
	if !ok {
		// clone
		_, err := p.gitea.EnsureRepo(ctx, org, repo)
		if err != nil {
			return nil, err
		}
		cloneurl := p.GenerateCloneURL(ctx, ref)

		logger = logger.WithValues("url", cloneurl, "branch", branch)
		logger.Info("cloning")
		repository, err := git.CloneContext(ctx, memory.NewStorage(), memfs.New(), &git.CloneOptions{
			URL:           cloneurl,
			Auth:          p.auth,
			ReferenceName: plumbing.NewBranchReferenceName(branch),
			SingleBranch:  true,
			Tags:          git.NoTags,
		})
		if err != nil {
			logger.Error(err, "clone failed")
			if !errors.Is(err, git.NoMatchingRefSpecError{}) && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
				return nil, err
			}
			logger.Info("create new branch")
			// push new branch to remote
			// 如果服务端没有该分支，则创建一个空分支
			// git init
			repository, err = git.Init(memory.NewStorage(), memfs.New())
			if err != nil {
				return nil, err
			}
			// git remote add origin {clone-url}
			if _, err = repository.CreateRemote(&config.RemoteConfig{
				Name: git.DefaultRemoteName, URLs: []string{cloneurl},
			}); err != nil {
				return nil, err
			}
			// git checkout -b {branch}
			if err = repository.CreateBranch(&config.Branch{Name: branch, Remote: git.DefaultRemoteName}); err != nil {
				return nil, err
			}
			// 这里无法使用 git checkout {branch} 类似的操作，因为是空的repo，HEAD指针指向了一个不存在的 master 分支
			// 可能是个 go-git 的bug，因为使用 git checkout 是正常的。
			// 使用了手动更改HEAD指向我们的目标分支的方式间接实现了 checkout
			// HEAD-> heads/refs/{branch} -> nil
			reference := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(branch))
			if err = repository.Storer.SetReference(reference); err != nil {
				return nil, err
			}
			wt, err := repository.Worktree()
			if err != nil {
				return nil, err
			}
			// echo "init" > readme.md
			if err := util.WriteFile(wt.Filesystem, "README.md", []byte("init"), os.ModePerm); err != nil {
				return nil, err
			}
			// git add .
			if err := wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
				return nil, err
			}
			// git commit -m "init"
			if _, err := wt.Commit("init", &git.CommitOptions{Author: p.commiter(), Committer: p.commiter()}); err != nil {
				return nil, err
			}
			// git push
			if err := repository.PushContext(ctx, &git.PushOptions{Auth: p.auth}); err != nil {
				return nil, err
			}
		}
		tocacherepo := &Repository{
			repository: repository,
			cloneurl:   cloneurl,
			p:          p,
			branch:     branch,
			lastsync:   time.Now(),
		}
		p.caches.Store(target, tocacherepo)
		return tocacherepo, nil
	}

	// ok
	cachedrepository := obj.(*Repository)
	if cachedrepository.Expired() {
		logger.Info("expired refresh cache", "lastsync", cachedrepository.lastsync.Format(time.RFC3339))
		// git pull
		if err := cachedrepository.Pull(ctx); err != nil {
			logger.Error(err, "git pull")
		}
	}
	return cachedrepository, nil
}

func (p *SimpleLocalProvider) commiter() *object.Signature {
	return &object.Signature{
		Name:  p.committer.Name,
		Email: p.committer.Email,
		When:  time.Now(),
	}
}
