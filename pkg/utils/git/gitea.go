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
	"fmt"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type GiteaRemote struct {
	*gitea.Client
}

func (h *GiteaRemote) EnsureRepo(ctx context.Context, orgname, reponame string) (*gitea.Repository, error) {
	repo, resp, err := h.GetRepo(orgname, reponame)
	if err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return nil, err
		}
		// create a new repo
		org, resp, err := h.GetOrg(orgname)
		if err != nil {
			if resp == nil || resp.StatusCode != http.StatusNotFound {
				return nil, err
			}
			org, _, err = h.CreateOrg(gitea.CreateOrgOption{
				Name: orgname, FullName: orgname,
				Description: fmt.Sprintf("org for tenaut %s", orgname),
			})
			if err != nil {
				return nil, err
			}
		}

		repo, _, err = h.CreateOrgRepo(org.FullName, gitea.CreateRepoOption{
			Name:        reponame,
			Description: fmt.Sprintf("repo for project [%s]", reponame),
		})
		if err != nil {
			return nil, err
		}
		return repo, nil
	}
	return repo, nil
}
