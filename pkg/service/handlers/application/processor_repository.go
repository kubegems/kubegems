package application

import (
	"context"

	"github.com/go-git/go-billy/v5"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/git"
)

type Repository struct {
	repo *git.Repository
	path string
}

func (r *Repository) Diff(ctx context.Context, hash string) ([]git.FileDiff, error) {
	return r.repo.Diff(ctx, r.path, hash)
}

func (r *Repository) HistoryFiles(ctx context.Context, hash string) (*git.Commit, error) {
	return r.repo.HistoryFiles(ctx, r.path, hash)
}

func (r *Repository) HistoryFunc(ctx context.Context, fun git.ContentVistitFunc) error {
	return r.repo.HistoryFunc(ctx, r.path, fun)
}

func (r *Repository) FS(ctx context.Context) (billy.Filesystem, error) {
	return r.repo.Filesystem(ctx, r.path)
}

type RepositoryFunc func(ctx context.Context, repository Repository) error

// using ContentFunc StoreFunc instead of directly calling
func (h *ManifestProcessor) Func(ctx context.Context, ref PathRef, funcs ...RepositoryFunc) error {
	gitref := ref.GitRef()
	gitrepo, err := h.GitProvider.Get(ctx, gitref)
	if err != nil {
		log.FromContextOrDiscard(ctx).Error(err, "get repository")
		return err
	}
	repo := &Repository{path: gitref.Path, repo: gitrepo}

	for _, f := range funcs {
		if err := f(ctx, *repo); err != nil {
			return err
		}
	}
	return nil
}

type RepositoryFileSystemFunc func(ctx context.Context, fs billy.Filesystem) error

func FsFunc(funcs ...RepositoryFileSystemFunc) RepositoryFunc {
	return func(ctx context.Context, repository Repository) error {
		fs, err := repository.FS(ctx)
		if err != nil {
			return err
		}
		for _, f := range funcs {
			if err := f(ctx, fs); err != nil {
				return err
			}
		}
		return nil
	}
}

func Commit(msg string) RepositoryFunc {
	return func(ctx context.Context, repository Repository) error {
		// commit
		if msg == "" {
			return nil
		}
		cm := &git.CommitMessage{
			Message:   msg,
			Committer: AuthorFromContext(ctx),
		}
		return repository.repo.CommitPushWithRetry(ctx, repository.path, cm)
	}
}

func UpdateKustomizeCommit(msg string) RepositoryFunc {
	return func(ctx context.Context, repository Repository) error {
		// update kustomization
		fs, err := repository.FS(ctx)
		if err != nil {
			return err
		}
		if err := InitOrUpdateKustomization(fs, nil, nil); err != nil {
			return err
		}
		// commit
		return Commit(msg)(ctx, repository)
	}
}

type GitStore = *FsStore

func FSStoreFunc(funcs ...func(ctx context.Context, store GitStore) error) RepositoryFileSystemFunc {
	return (func(ctx context.Context, fs billy.Filesystem) error {
		store := NewGitFsStore(fs)
		for _, f := range funcs {
			if err := f(ctx, store); err != nil {
				return err
			}
		}
		return nil
	})
}
