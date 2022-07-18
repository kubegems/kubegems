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

package harbor

import (
	"testing"
)

func TestParseImag(t *testing.T) {
	type args struct {
		image string
	}
	tests := []struct {
		name         string
		args         args
		wantDomain   string
		wantProject  string
		wantArtifact string
		wantTag      string
		wantErr      bool
	}{
		{
			name: "noarmal",
			args: args{
				image: "harbor.foo.com/project/artifact:tag",
			},
			wantDomain:   "harbor.foo.com",
			wantProject:  "project",
			wantArtifact: "artifact",
			wantTag:      "tag",
		},
		{
			name: "nodomain",
			args: args{
				image: "project/artifact:tag",
			},
			wantDomain:   "docker.io",
			wantProject:  "project",
			wantArtifact: "artifact",
			wantTag:      "tag",
		},
		{
			name: "t1",
			args: args{
				image: "project/foo/artifact:tag",
			},
			wantDomain:   "docker.io",
			wantProject:  "project",
			wantArtifact: "foo/artifact",
			wantTag:      "tag",
		},
		{
			name: "sha256",
			args: args{
				image: "harbor.foo.com/foo/bar@sha256:e5c220e2c9d52289682cb8544aed260d6b07900c9a525853507b3303224b9e23",
			},
			wantDomain:   "harbor.foo.com",
			wantProject:  "foo",
			wantArtifact: "bar",
			wantTag:      "sha256:e5c220e2c9d52289682cb8544aed260d6b07900c9a525853507b3303224b9e23",
		},
		{
			name: "nolib",
			args: args{
				image: "harbor.foo.com/foo:tag",
			},
			wantDomain:   "harbor.foo.com",
			wantProject:  "library",
			wantArtifact: "foo",
			wantTag:      "tag",
		},
		{
			name: "notag",
			args: args{
				image: "harbor.foo.com/foo",
			},
			wantDomain:   "harbor.foo.com",
			wantProject:  "library",
			wantArtifact: "foo",
			wantTag:      "latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDomain, gotProject, gotArtifact, gotTag, err := ParseImag(tt.args.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseImag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotDomain != tt.wantDomain {
				t.Errorf("ParseImag() gotDomain = %v, want %v", gotDomain, tt.wantDomain)
			}
			if gotProject != tt.wantProject {
				t.Errorf("ParseImag() gotProject = %v, want %v", gotProject, tt.wantProject)
			}
			if gotArtifact != tt.wantArtifact {
				t.Errorf("ParseImag() gotArtifact = %v, want %v", gotArtifact, tt.wantArtifact)
			}
			if gotTag != tt.wantTag {
				t.Errorf("ParseImag() gotTag = %v, want %v", gotTag, tt.wantTag)
			}
		})
	}
}

func TestTrimImageTag(t *testing.T) {
	type args struct {
		image string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want2 string
	}{
		{
			name: "normal",
			args: args{
				image: "harbor.foo.com/library/nginx:v1.4.13",
			},
			want:  "harbor.foo.com/library/nginx",
			want2: "v1.4.13",
		},
		{
			name: "normal2",
			args: args{
				image: "library/nginx:v1.4.13",
			},
			want:  "library/nginx",
			want2: "v1.4.13",
		},
		{
			name: "normal no domain",
			args: args{
				image: "nginx:v1.4.13",
			},
			want:  "nginx",
			want2: "v1.4.13",
		},
		{
			name: "normal no tag",
			args: args{
				image: "harbor.foo.com/library/nginx",
			},
			want:  "harbor.foo.com/library/nginx",
			want2: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, got2 := SplitImageNameTag(tt.args.image); got != tt.want || got2 != tt.want2 {
				t.Errorf("TrimImageTag() = %v,%v, want %v,%v", got, got2, tt.want, tt.want2)
			}
		})
	}
}
