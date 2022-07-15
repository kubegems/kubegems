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
