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
