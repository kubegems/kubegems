package utils

import (
	"bufio"
	"os"
)

func EnsurePathExists(dirname string) error {
	_, err := os.Stat(dirname)
	if os.IsExist(err) {
		return nil
	}
	return os.MkdirAll(dirname, os.FileMode(0644))
}

func CopyFileByLine(dest, src string) (lineCount int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()
	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
	if err != nil {
		return
	}
	defer destFile.Close()

	lineCount = 0
	scanner := bufio.NewScanner(srcFile)
	for scanner.Scan() {
		_, err = destFile.WriteString(scanner.Text() + "\n")
		if err != nil {
			return
		}
		lineCount++
	}
	return
}
