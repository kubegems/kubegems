// Copyright 2023 The kubegems.io Authors
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

package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	BaseDir  string `json:"baseDir"`
	Mode     string `json:"mode"`
	Owner    string `json:"owner"`
	Group    string `json:"group"`
	Size     int64  `json:"size"`
	ModTime  string `json:"modtime"`
	Name     string `json:"name"`
	Nlink    uint64 `json:"nlink"`
	IsLink   bool   `json:"islink"`
	IsDir    bool   `json:"isDir"`
	RealFile string `json:"realFile"`
}

func (f *FileInfo) ReadLinkRealPath() {
	if f.IsLink {
		realPath, err := filepath.EvalSymlinks(path.Join(f.BaseDir, f.Name))
		if err == nil {
			f.RealFile = realPath
		}
	}
}

type Files []*FileInfo

func (f Files) Show() {
	s, _ := json.Marshal(f)
	os.Stdout.Write(s)
	// fmt.Println(string(s))
}

func ListDir(target string) (Files, error) {
	stat, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return []*FileInfo{toFileInfo(stat, path.Base(target))}, nil
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}
	infos := make([]fs.FileInfo, len(entries))
	for idx, entry := range entries {
		f, e := entry.Info()
		if e != nil {
			return nil, e
		}
		infos[idx] = f
	}
	ret := []*FileInfo{}
	if strings.HasSuffix(target, string(os.PathSeparator)) {
		if target != "/" {
			target = strings.TrimRight(target, "/")
		}
	}
	for _, fileInfo := range infos {
		ret = append(ret, toFileInfo(fileInfo, target))
	}
	return ret, nil
}
