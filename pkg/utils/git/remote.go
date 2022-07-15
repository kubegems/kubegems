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

package git

import (
	"context"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RevMeta struct {
	Hash    string
	Author  string
	Date    metav1.Time
	Message string
}

func (p *SimpleLocalProvider) GetRemoteRepoRevMeta(ctx context.Context, repourl string, branchOrRev string) (*RevMeta, error) {
	// clone now
	repository, err := git.CloneContext(ctx, memory.NewStorage(), nil, &git.CloneOptions{
		URL:  repourl,
		Auth: p.auth,
	})
	if err != nil {
		return nil, err
	}

	// branch need remote update
	hash, err := repository.ResolveRevision(plumbing.Revision(branchOrRev))
	if err != nil {
		// if not found
		// remote update
		return nil, err
	}

	commit, err := repository.CommitObject(*hash)
	if err != nil {
		return nil, err
	}
	return &RevMeta{
		Author:  commit.Author.Name,
		Date:    metav1.NewTime(commit.Author.When),
		Message: commit.Message,
		Hash:    hash.String(),
	}, nil
}
