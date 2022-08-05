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
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func ExampleServer_Run() {
	ctx := context.Background()
	url := "http://s3.example.com"
	s3opts := &S3ContentManagerOptions{
		URL:    url,
		Bucket: "git-lfs",
		Credential: aws.Credentials{
			AccessKeyID:     "",
			SecretAccessKey: "",
		},
		LinkExpireIn: time.Hour,
	}
	s3lfs, err := NewS3ContentManager(ctx, s3opts)
	if err != nil {
		panic(err)
	}
	opts := NewDefaultOptions()
	opts.UseGitHTTPBackend = true

	s := Server{
		GitBase: "/var/www/git",
		LFS:     s3lfs,
	}
	if err := s.Run(ctx, opts); err != nil {
		panic(err)
	}
}
