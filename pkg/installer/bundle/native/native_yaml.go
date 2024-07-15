// Copyright 2024 The kubegems.io Authors
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

package native

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
)

func NewNativeFunc(ctx context.Context, bundle *pluginsv1beta1.Plugin, dir string) ([]byte, error) {
	// read all yaml files in dir
	ret := []byte{}

	vfs := os.DirFS(dir)
	err := fs.WalkDir(vfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		f, err := vfs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		// read file
		file, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		ret = append(ret, []byte("---\n")...)
		ret = append(ret, file...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
