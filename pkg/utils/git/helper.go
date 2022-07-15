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
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-billy/v5/util"
)

// ForFileNameFunc filename 为 base 下的全路径
func ForFileNameFunc(fs billy.Filesystem, base string, fun func(filename string) error) error {
	if base != "" && base != "." {
		fs = chroot.New(fs, base)
	}
	return forFileNameFunc(fs, "", fun)
}

func forFileNameFunc(fs billy.Filesystem, base string, fun func(filename string) error) error {
	stat, err := fs.Stat(base)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		fis, err := fs.ReadDir(base)
		if err != nil {
			return err
		}
		for _, fi := range fis {
			if err := forFileNameFunc(fs, filepath.Join(base, fi.Name()), fun); err != nil {
				if err == filepath.SkipDir {
					return nil
				}
				return err
			}
		}
		return nil
	}

	return fun(base)
}

func ForFileContentFunc(fs billy.Filesystem, base string, fun func(filename string, content []byte) error) error {
	return ForFileNameFunc(fs, base, func(filename string) error {
		bts, err := util.ReadFile(fs, filepath.Join(base, filename))
		if err != nil {
			return err
		}
		if err := fun(filename, bts); err != nil {
			return err
		}
		return nil
	})
}
