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
	"log"
	"os"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
)

func TestForFileNameFunc(t *testing.T) {
	type args struct {
		fs   billy.Filesystem
		base string
		fun  func(filename string) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				fs: func() billy.Filesystem {
					fs := memfs.New()
					util.WriteFile(fs, "a/b.txt", []byte("hello"), os.ModePerm)
					util.WriteFile(fs, "a/b/c.txt", []byte("hello"), os.ModePerm)
					util.WriteFile(fs, "a/b/c2.txt", []byte("hello"), os.ModePerm)
					util.WriteFile(fs, "a/b2.txt", []byte("hello"), os.ModePerm)
					util.WriteFile(fs, "a/b3.txt", []byte("hello"), os.ModePerm)
					util.WriteFile(fs, "a2.txt", []byte("hello"), os.ModePerm)
					return fs
				}(),
				base: "a",
				fun: func(filename string) error {
					log.Printf(filename)
					return nil
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ForFileNameFunc(tt.args.fs, tt.args.base, tt.args.fun); (err != nil) != tt.wantErr {
				t.Errorf("forFileNameFunc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestForFileContentFunc(t *testing.T) {
	type args struct {
		fs   billy.Filesystem
		base string
		fun  func(filename string, content []byte) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				fs: func() billy.Filesystem {
					fs := memfs.New()
					util.WriteFile(fs, "a/b.txt", []byte("a-b"), os.ModePerm)
					util.WriteFile(fs, "a/b/c.txt", []byte("a-b-c"), os.ModePerm)
					util.WriteFile(fs, "a/b/c2.txt", []byte("a-b-c2"), os.ModePerm)
					util.WriteFile(fs, "a/b2.txt", []byte("a-b2"), os.ModePerm)
					util.WriteFile(fs, "a/b3.txt", []byte("a-b3"), os.ModePerm)
					util.WriteFile(fs, "a2.txt", []byte("a2"), os.ModePerm)
					return fs
				}(),
				base: "a/b.txt",
				fun: func(filename string, content []byte) error {
					log.Printf("file %s=[%s]", filename, content)
					return nil
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ForFileContentFunc(tt.args.fs, tt.args.base, tt.args.fun); (err != nil) != tt.wantErr {
				t.Errorf("ForFileContentFunc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
