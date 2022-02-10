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
