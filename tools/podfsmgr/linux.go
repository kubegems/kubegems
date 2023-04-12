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

//go:build !windows

package main

import (
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func toFileInfo(fileInfo fs.FileInfo, basedir string) *FileInfo {
	statT := fileInfo.Sys().(*syscall.Stat_t)
	finfo := &FileInfo{
		BaseDir: basedir,
		Mode:    fileInfo.Mode().String(),
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime().String(),
		Name:    fileInfo.Name(),
		IsLink:  fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink,
		IsDir:   fileInfo.IsDir(),
		Nlink:   uint64(statT.Nlink),
	}
	finfo.ReadLinkRealPath()
	uid := strconv.FormatUint(uint64(statT.Uid), 10)
	gid := strconv.FormatUint(uint64(statT.Gid), 10)
	owner, err := user.LookupId(uid)
	if err != nil {
		finfo.Owner = uid
	} else {
		finfo.Owner = owner.Username
	}
	group, err := user.LookupGroupId(gid)
	if err != nil {
		finfo.Group = gid
	} else {
		finfo.Group = group.Name
	}
	return finfo
}
