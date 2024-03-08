package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/library/rest/request"
)

type RemoteClient struct {
	Address string
	client  *http.Client
}

var _ Client = &RemoteClient{}

func NewDefaultRemoteClient() *RemoteClient {
	return NewRemoteClient("http://kubegems-worker.kubegems")
}

func NewRemoteClient(address string) *RemoteClient {
	return &RemoteClient{Address: address, client: http.DefaultClient}
}

// ListTasks implements Client.
func (r *RemoteClient) ListTasks(ctx context.Context, group string, name string) ([]Task, error) {
	var tasks []Task
	q := map[string]string{"group": group, "name": name}
	if err := r.do(ctx, http.MethodGet, "/tasks", q, nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// RemoveTask implements Client.
func (r *RemoteClient) RemoveTask(ctx context.Context, group string, name string, uid string) error {
	return r.do(ctx, http.MethodDelete, "/tasks", nil, Task{Group: group, Name: name, UID: uid}, nil)
}

// SubmitTask implements Client.
func (r *RemoteClient) SubmitTask(ctx context.Context, task Task) error {
	return r.do(ctx, http.MethodPost, "/tasks", nil, task, nil)
}

// WatchTasks implements Client.
func (r *RemoteClient) WatchTasks(ctx context.Context, group string, name string, onchange func(ctx context.Context, task *Task) error) error {
	q := map[string]string{"group": group, "name": name, "watch": "true"}
	resp, err := r.doraw(ctx, http.MethodGet, "/tasks", q, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			task := &Task{}
			if err := dec.Decode(task); err != nil {
				return fmt.Errorf("failed to decode task: %w", err)
			}
			if err := onchange(ctx, task); err != nil {
				return err
			}
		}
	}
}

func (r *RemoteClient) doraw(ctx context.Context, method string, path string, queries map[string]string, body any) (*http.Response, error) {
	var bodyreader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyreader = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequest(method, r.Address+path, bodyreader)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	q := req.URL.Query()
	for k, v := range queries {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	return r.client.Do(req)
}

func (r *RemoteClient) do(ctx context.Context, method string, path string, queries map[string]string, body any, into any) error {
	resp, err := r.doraw(ctx, method, path, queries, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	if into != nil {
		if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
			return err
		}
	}
	return nil
}

type RemoteClientServer struct {
	Client Client
}

func NewRemoteClientServer(client Client) *RemoteClientServer {
	return &RemoteClientServer{Client: client}
}

func (s *RemoteClientServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		log.Info("request", "method", r.Method, "url", r.URL.String())
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("watch") == "true" {
				s.watch(w, r)
			} else {
				s.list(w, r)
			}
		case http.MethodPost:
			s.submit(w, r)
		case http.MethodDelete:
			s.remove(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	return mux
}

func (s *RemoteClientServer) submit(w http.ResponseWriter, r *http.Request) {
	task := Task{}
	if err := request.Body(r, &task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Client.SubmitTask(r.Context(), task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *RemoteClientServer) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	group, name := q.Get("group"), q.Get("name")
	tasks, err := s.Client.ListTasks(r.Context(), group, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *RemoteClientServer) remove(w http.ResponseWriter, r *http.Request) {
	task := Task{}
	if err := request.Body(r, &task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Client.RemoveTask(r.Context(), task.Group, task.Name, task.UID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *RemoteClientServer) watch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	group, name := q.Get("group"), q.Get("name")

	enc := json.NewEncoder(w)
	if err := s.Client.WatchTasks(r.Context(), group, name, func(ctx context.Context, task *Task) error {
		if err := enc.Encode(task); err != nil {
			return err
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
