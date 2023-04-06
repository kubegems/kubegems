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
