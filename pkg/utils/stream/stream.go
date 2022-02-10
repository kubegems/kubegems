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
