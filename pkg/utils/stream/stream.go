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

package stream

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type WriterFlusher interface {
	http.Flusher
	io.Writer
}

type Pusher struct {
	encoder *json.Encoder
	dst     WriterFlusher
}

func StartPusher(w http.ResponseWriter) (*Pusher, error) {
	flusher, ok := w.(WriterFlusher)
	if !ok {
		return nil, errors.New("not a flushable writer")
	}
	// before we start push ,must send response headers first
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "application/json")
	// send ok status

	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &Pusher{dst: flusher, encoder: json.NewEncoder(flusher)}, nil
}

func (p *Pusher) Push(data interface{}) error {
	if err := p.encoder.Encode(data); err != nil {
		return err
	}
	p.dst.Flush()
	return nil
}

type Receiver struct {
	decoder *json.Decoder
}

func StartReceiver(src io.Reader) *Receiver {
	return &Receiver{decoder: json.NewDecoder(src)}
}

func (r *Receiver) Recieve(into interface{}) error {
	return r.decoder.Decode(into)
}
