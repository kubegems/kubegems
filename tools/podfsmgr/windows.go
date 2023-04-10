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

//go:build windows

// not tested

package main

import (
	"io/fs"
	"os"
)

func toFileInfo(fileInfo fs.FileInfo, basedir string) *FileInfo {
	finfo := &FileInfo{
		BaseDir: basedir,
		Mode:    fileInfo.Mode().String(),
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime().String(),
		Name:    fileInfo.Name(),
		IsLink:  fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink,
		IsDir:   fileInfo.IsDir(),
	}
	finfo.ReadLinkRealPath()
	return finfo
}
