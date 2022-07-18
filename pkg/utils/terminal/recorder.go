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

package terminal

import (
	"bytes"
	"fmt"
)

func NewTerminalRecorder() *TerminalRecorder {
	return &TerminalRecorder{
		buf: bytes.NewBuffer([]byte{}),
	}
}

type TerminalRecorder struct {
	buf *bytes.Buffer
}

func (t *TerminalRecorder) Write(data []byte) (int, error) {
	/* TODO
	length := len(data) + t.buf.Len()
	left := length % 1024
	step := length / 1024
	for idx := 0; idx < step; idx ++ {

	}


	*/
	return t.buf.Write(data)
}

func (t *TerminalRecorder) Close() error {
	if t.buf.Len() > 0 {
		t.flush()
	}
	return nil
}

func (t *TerminalRecorder) flush() {
	// TODO: WREITE FILE
	fmt.Println(t.buf.String())
	t.buf.Reset()
}
