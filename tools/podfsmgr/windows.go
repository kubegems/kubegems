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
