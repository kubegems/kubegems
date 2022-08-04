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
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestS3ContentManager_Upload(t *testing.T) {
	ctx := context.Background()

	m, err := NewS3ContentManager(ctx, &S3ContentManagerOptions{
		URL:    "https://s3.amazonaws.com",
		Bucket: "git-lfs",
		Credential: aws.Credentials{
			AccessKeyID:     "",
			SecretAccessKey: "",
		},
		LinkExpireIn: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.Upload(ctx, "user/repo", "123")
	if err != nil {
		t.Errorf("S3ContentManager.Upload() error = %v", err)
		return
	}
	log.Print(got)
}
