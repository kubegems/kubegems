package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubegems/gems/pkg/log"
	"github.com/opentracing/opentracing-go"
)

type Repository struct {
	mu         sync.Mutex
	p          *SimpleLocalProvider
	branch     string
	cloneurl   string
	lastsync   time.Time
	repository *git.Repository

	pulllock *pulllock // for concurrent pulling
}

func (r *Repository) CloneURL() string { return r.cloneurl }

func (r *Repository) Pull(ctx context.Context) error {
	if r.pulllock == nil {
		r.pulllock = &pulllock{}
	}
	return r.pulllock.Go(func() error {
		return r.resetOriginLatest(ctx, git.HardReset)
	})
}

func (r *Repository) resetOriginLatest(ctx context.Context, resetmode git.ResetMode) error {
	log.FromContextOrDiscard(ctx).Info("pulling repository", "repository", r.cloneurl, "lastsync", r.lastsync)

	r.mu.Lock()
	defer r.mu.Unlock()

	// git remote update
	if err := r.repository.FetchContext(ctx, &git.FetchOptions{Auth: r.p.auth, Force: true}); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return fmt.Errorf("failed to fetch remote: %w", err)
		}
		r.lastsync = time.Now()
		return nil
	}
	hash, err := r.repository.ResolveRevision(plumbing.Revision(fmt.Sprintf("refs/remotes/origin/%s", r.branch)))
	if err != nil {
		return fmt.Errorf("failed to resolve repository: %w", err)
	}
	// git reset origin branch --hard
	wt, err := r.repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}
	if err := wt.Reset(&git.ResetOptions{Commit: *hash, Mode: resetmode}); err != nil {
		return fmt.Errorf("failed to reset repository: %w", err)
	}

	r.lastsync = time.Now()
	return nil
}

func (r *Repository) Expired() bool {
	return r.lastsync.Add(r.p.cacheexpire).Before(time.Now())
}

type FileDiff struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
}

func (r *Repository) Diff(ctx context.Context, path string, hash string) ([]FileDiff, error) {
	log.FromContextOrDiscard(ctx).Info("diff", "path", path, "hash", hash)

	r.mu.Lock()
	defer r.mu.Unlock()

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	acommit, err := r.repository.CommitObject(plumbing.NewHash(hash))
	if err != nil {
		return nil, err
	}
	parent, err := acommit.Parent(0)
	if err != nil {
		return nil, err
	}
	patch, err := parent.PatchContext(ctx, acommit)
	if err != nil {
		return nil, err
	}

	diffs := []FileDiff{}
	for _, patch := range patch.FilePatches() {
		if patch.IsBinary() {
			continue
		}
		from, to := patch.Files()

		filename := ""
		if from != nil {
			filename = from.Path()
		}
		if to != nil {
			filename = to.Path()
		}

		if !strings.HasPrefix(filename, path) {
			continue
		}

		basedfilename := strings.TrimPrefix(filename, path)
		fromcontent, _ := readDiffFileContent(r, from)
		tocontent, _ := readDiffFileContent(r, to)

		diffs = append(diffs, FileDiff{
			Name: basedfilename,
			From: string(fromcontent),
			To:   string(tocontent),
		})
	}
	return diffs, nil
}

func readDiffFileContent(repo *Repository, f diff.File) ([]byte, error) {
	if f == nil {
		return nil, fmt.Errorf("null file")
	}
	blob, err := repo.repository.BlobObject(f.Hash())
	if err != nil {
		return nil, err
	}
	rc, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return content, nil
}

type CommitFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Commit struct {
	Author    object.Signature `json:"author"`
	Message   string           `json:"message"`
	Hash      string           `json:"hash"`
	Committer object.Signature `json:"committer"`
	Files     []CommitFile     `json:"files"`
}

type ContentVistitFunc func(ctx context.Context, commit Commit) error

func (r *Repository) HistoryFunc(ctx context.Context, path string, fun ContentVistitFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	iter, err := r.repository.Log(&git.LogOptions{
		PathFilter: func(s string) bool { return strings.HasPrefix(s, path) },
	})
	if err != nil {
		return err
	}
	return iter.ForEach(func(c *object.Commit) error {
		return fun(ctx, Commit{
			Hash:      c.Hash.String(),
			Message:   c.Message,
			Author:    c.Author,
			Committer: c.Committer,
		})
	})
}

type ContentUpdateFunc = func(ctx context.Context, fs billy.Filesystem) error

func (r *Repository) Filesystem(ctx context.Context, path string) (billy.Filesystem, error) {
	// 获取当前的 worktree
	wt, err := r.repository.Worktree()
	if err != nil {
		return nil, err
	}
	// precheck
	if err := wt.Filesystem.MkdirAll(path, os.ModePerm); err != nil {
		return nil, err
	}
	appfs := wt.Filesystem
	if path != "" {
		appfs = chroot.New(appfs, path)
	}
	return appfs, nil
}

// ContentFunc change fs content and do a commit if commit is not nil
func (r *Repository) ContentFunc(ctx context.Context, path string, fun ContentUpdateFunc) error {
	fs, err := r.Filesystem(ctx, path)
	if err != nil {
		return err
	}
	// 更新 git 内容
	if err := fun(ctx, fs); err != nil {
		return err
	}
	return nil
}

type CommitMessage struct {
	Message   string
	Committer *object.Signature
}

const MaxCommitRetry = 3

func (r *Repository) CommitPushWithRetry(ctx context.Context, path string, commit *CommitMessage) error {
	var err error
	log := log.FromContextOrDiscard(ctx)

	defer func() {
		if err != nil {
			err = fmt.Errorf("commit failed: %w", err)
			log.Error(err, "commit push failed")
		}
	}()

	for i := 0; i <= MaxCommitRetry; i++ {
		if err = r.CommitPush(ctx, path, commit); err != nil {
			// non-fast-forward error
			log.Error(err, "commit push failed retry", "i", i)
			if isNonFastForwardError(err) {
				// try git pull reset
				if err := r.resetOriginLatest(ctx, git.MixedReset); err != nil {
					return err
				}
				continue
			}
			// other err
			return err
		}
		// no err
		return nil
	}
	return err
}

func (r *Repository) CommitPush(ctx context.Context, path string, commit *CommitMessage) error {
	log.FromContextOrDiscard(ctx).Info("commit push", "path", path)

	if commit == nil || len(commit.Message) == 0 {
		return nil // do not commit
	}
	span, ctx := opentracing.StartSpanFromContext(ctx, "commit-push")
	defer span.Finish()

	r.mu.Lock()
	defer r.mu.Unlock()

	// 获取当前的 worktree
	wt, err := r.repository.Worktree()
	if err != nil {
		return err
	}

	// is no changes
	status, err := wt.Status()
	if err != nil {
		return fmt.Errorf("unable to get wt status: %w", err)
	}
	if !status.IsClean() {
		log.FromContextOrDiscard(ctx).Info("wotktree not clean do commit", "path", path)
		// add 删除的文件会失败 https://github.com/go-git/go-git/pull/242
		// 如果该bug修好了就可以改为
		// if err := wt.AddWithOptions(&git.AddOptions{Path: ref.Path}); err != nil {
		// return fmt.Errorf("failed to add worktree changes: %w", err)
		// }

		// git add {path}
		for filename, filestatus := range status {
			if !strings.HasPrefix(filename, path) {
				continue
			}
			if filestatus.Worktree == git.Deleted {
				if _, err := wt.Remove(filename); err != nil {
					return fmt.Errorf("failed to stash %s: %w", filename, err)
				}
			} else {
				if err := wt.AddWithOptions(&git.AddOptions{Path: filename}); err != nil {
					return fmt.Errorf("failed to stash %s: %w", filename, err)
				}
			}
		}

		// git commit -m {msg}
		if _, err := wt.Commit(commit.Message, &git.CommitOptions{
			Author:    &object.Signature{Name: commit.Committer.Name, Email: commit.Committer.Email, When: time.Now()},
			Committer: r.p.commiter(),
		}); err != nil {
			return fmt.Errorf("failed to commit %w", err)
		}
	}

	// git push origin {branch}
	if err := r.repository.PushContext(ctx, &git.PushOptions{Auth: r.p.auth}); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return err
	}
	return nil
}

func isNonFastForwardError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, git.ErrNonFastForwardUpdate) {
		return true
	}
	return strings.Contains(err.Error(), "non-fast-forward update:")
}

func (r *Repository) HistoryFiles(ctx context.Context, path string, rev string) (*Commit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	hash, err := r.repository.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return nil, err
	}
	commit, err := r.repository.CommitObject(*hash)
	if err != nil {
		return nil, err
	}

	tree, err := r.repository.TreeObject(commit.TreeHash)
	if err != nil {
		return nil, err
	}
	basepathedtree, err := tree.Tree(path)
	if err != nil {
		return nil, err
	}

	commitefiles := []CommitFile{}
	treeiter := basepathedtree.Files()
	for {
		file, err := treeiter.Next()
		if err != nil {
			break //
		}
		content, err := file.Contents()
		if err != nil {
			continue
		}
		commitefiles = append(commitefiles, CommitFile{
			Name:    file.Name,
			Content: content,
		})
	}
	ret := &Commit{
		Author:    commit.Author,
		Message:   commit.Message,
		Hash:      commit.Hash.String(),
		Committer: commit.Committer,
		Files:     commitefiles,
	}
	return ret, nil
}

type pulllock struct {
	lock       sync.Mutex
	rw         sync.RWMutex
	processing bool
	err        error
}

// Wait blocks until the lock is released.
// 使用场景:
// 目前使用在 git pull 并发场景，多个 git pull 时，后续的请求会等待第一个 git pull 完成并使用其返回值作为返回值。
// 以避免段时间内排队多次重复pull
func (l *pulllock) Go(fun func() error) error {
	l.lock.Lock() // 第一层锁用于锁住 processing 状态，当为false时，需要锁至其值改变为 true 时，即保证只有一个routine拿到false。
	processing := l.processing
	if processing {
		l.lock.Unlock()

		l.rw.RLock() // rw 锁会在第一个routine(拿到false)运行时锁住，其他routine(true) 持 rw.R 在此等待 rw.Unlock()。
		defer l.rw.RUnlock()
		return l.err
	} else {
		l.processing = true
		l.rw.Lock() // rw 锁需要在其他routine进入 rw.R 前锁住，否则其他routine会直接返回。

		l.lock.Unlock()

		l.err = fun()
		l.processing = false
		l.rw.Unlock()
	}
	return l.err
}
