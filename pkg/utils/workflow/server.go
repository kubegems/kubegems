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

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/retry"
)

const (
	DefaultTaskTimeout = 5 * time.Minute
)

type Options struct {
	Addr     string `json:"addr,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Server struct {
	backend    Backend
	registered map[string]interface{}
	executerid string
}

func NewServerFromRedisClient(cli *redis.Client) *Server {
	return NewServerFromBackend(NewRedisBackendFromClient(cli))
}

func NewServerFromBackend(backend Backend) *Server {
	executerid, _ := os.Hostname()
	return &Server{
		backend:    backend,
		registered: map[string]interface{}{},
		executerid: executerid,
	}
}

func NewServer(options *Options) (*Server, error) {
	backend := NewRedisBackend(options.Addr, options.Username, options.Password)
	return NewServerFromBackend(backend), nil
}

func (s *Server) NewClient(ctx context.Context) *Client {
	return NewClientFromBackend(s.backend)
}

func (s *Server) Run(ctx context.Context) error {
	log := log.FromContextOrDiscard(ctx)
	// consume submit queue
	return retry.OnError(retry.NotContextCancelError, func() error {
		log.Info("starting work consumer...")
		if err := s.backend.Sub(ctx, "submit", s.consume, WithConcurrency(5), WithAutoACK(true)); err != nil {
			log.Error(err, "subscripe failed, retry...")
			return err
		}
		return nil
	})
}

func (s *Server) consume(ctx context.Context, _ string, val []byte) error {
	log := log.FromContextOrDiscard(ctx)

	task := &jsonArgsTask{}
	if err := json.Unmarshal(val, task); err != nil {
		log.Error(err, "decode task")
		return nil // ignore error
	}
	log = log.WithValues("name", task.Name, "uid", task.UID)

	log.Info("consume task")
	ctx = logr.NewContext(ctx, log)

	finished := s.process(ctx, task)
	if !finished {
		// requeue updated task
		content, err := json.Marshal(task)
		if err != nil {
			return err
		}
		log.Info("requeue task")
		s.backend.Pub(ctx, "submit", "", content)
		return nil
	}
	log.Info("finished task")
	return nil
}

// ??????????????????????????????????????? task ???????????? stask ????????????
func (s *Server) process(ctx context.Context, task *jsonArgsTask) bool {
	// foreach task
	if task.UID == "" {
		task.UID = uuid.New().String()
	}
	if err := s.processone(ctx, task, task.Steps); err != nil {
		// ??????????????? ??????finished
		task.Status.FinishTimestamp = metav1.Now()
		task.Status.Status = TaskStatusError
		task.Status.Message = err.Error()
		_ = s.updateTask(ctx, task)
		return true
	} else if isAllFinished(task.Steps) {
		// ???????????????????????????????????? finished
		task.Status.FinishTimestamp = metav1.Now()
		task.Status.Status = TaskStatusSuccess
		_ = s.updateTask(ctx, task)
		return true
	} else {
		// ???????????????????????????????????????????????????
		return false
	}
}

func isAllFinished(steps []*jsonArgsStep) bool {
	for _, step := range steps {
		if step.Status.Status != TaskStatusSuccess {
			return false
		}
		if !isAllFinished(step.SubSteps) {
			return false
		}
	}
	return true
}

func (s *Server) processone(ctx context.Context, task *jsonArgsTask, steps []*jsonArgsStep) error {
	// ?????????value???context
	ctx = WithValues(ctx, task.Addtionals)

	for _, step := range steps {
		switch step.Status.Status {
		case "", TaskStatusRunning:
			// save init state
			step.Status = TaskStatus{
				Status:         TaskStatusRunning,
				StartTimestamp: metav1.Now(),
				Executer:       s.executerid,
			}

			_ = s.updateTask(ctx, task)
			if step.Function != "" {
				if err := s.execute(ctx, step); err != nil {
					step.Status.Status = TaskStatusError
					step.Status.Message = err.Error()
					// ???????????????????????????
					step.Status.FinishTimestamp = metav1.Now()
					_ = s.updateTask(ctx, task)
					return err
				} else {
					step.Status.Status = TaskStatusSuccess
					step.Status.FinishTimestamp = metav1.Now()
					_ = s.updateTask(ctx, task)
					// ??????step???????????????????????? nil ????????????
					// ???????????????nil??????????????????????????????????????????
					return nil
				}
			} else {
				// ???????????????????????????????????????????????????????????????
				step.Status.FinishTimestamp = metav1.Now()
				step.Status.Status = TaskStatusSuccess
			}
		case TaskStatusError:
			return errors.New(step.Status.Message) // ???????????????????????????????????????????????????
		}
		// ?????? substeps
		if err := s.processone(ctx, task, step.SubSteps); err != nil {
			return err
		}
	}
	return nil
}

func (n *Server) updateTask(ctx context.Context, task *jsonArgsTask) error {
	content, err := json.Marshal(task)
	if err != nil {
		return err
	}
	taskjkey := strings.Join([]string{task.Group, task.Name, task.UID}, "/")
	return n.backend.Put(ctx, taskjkey, content, 5*time.Minute)
}

func (n *Server) Register(name string, fun interface{}) error {
	// validate
	t := reflect.ValueOf(fun).Type()
	if t.Kind() != reflect.Func {
		return fmt.Errorf("name [%s] fun [%v] not a function", name, fun)
	}
	// register
	if _, ok := n.registered[name]; ok {
		return fmt.Errorf("name [%s] fun [%v] already registered", name, fun)
	}
	n.registered[name] = fun
	return nil
}

func (n *Server) execute(ctx context.Context, task *jsonArgsStep) (err error) {
	if task.Timeout == 0 {
		task.Timeout = DefaultTaskTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	log := log.FromContextOrDiscard(ctx)
	log.Info("executing", "step", task.Name, "func", task.Function, "args", task.Args)

	name := task.Function
	fun, ok := n.registered[name]
	if !ok {
		return fmt.Errorf("func %s not registered", name)
	}

	defer func() {
		if e := recover(); e != nil {
			log.Info("executed panic", "step", task.Name, "func", task.Function, "err", e)
			switch e := e.(type) {
			default:
				err = errors.New("failed to execute")
			case error:
				err = e
			case string:
				err = errors.New(e)
			}
		}
	}()

	funv := reflect.ValueOf(fun)
	funt := funv.Type()

	argsv := []reflect.Value{}
	argsi := 0
	for i := 0; i < funt.NumIn(); i++ {
		argt := funt.In(i)

		// ???????????????????????? context.Context ????????????context????????????
		if i == 0 && argt.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			argsv = append(argsv, reflect.ValueOf(ctx))
			continue
		}

		arg := reflect.New(argt).Interface()

		// ????????????????????????????????????????????????
		if argsi < len(task.Args) {
			if err := json.Unmarshal(task.Args[argsi], &arg); err != nil {
				return err
			}
		}
		// ????????????????????????

		argsv = append(argsv, reflect.Indirect(reflect.ValueOf(arg)))
		argsi++
	}

	// execute
	var rvs []reflect.Value
	if funt.IsVariadic() {
		rvs = funv.CallSlice(argsv)
	} else {
		rvs = funv.Call(argsv)
	}
	// ?????????????????????
	for _, result := range rvs {
		task.Status.Result = append(task.Status.Result, reflect.Indirect(result).Interface())
	}
	log.Info("executed", "step", task.Name, "func", task.Function, "result", task.Status.Result)
	// ???????????????????????????????????? error ???????????????error
	if e, ok := rvs[len(rvs)-1].Interface().(error); ok {
		err = e
	}
	return err
}

func ValueFromConetxt(ctx context.Context, key string) string {
	if val, ok := ctx.Value(key).(string); ok {
		return val
	}
	return ""
}

func WithValues(ctx context.Context, kvs map[string]string) context.Context {
	return &RuntimeValuesContext{parent: ctx, kvs: kvs}
}

type RuntimeValuesContext struct {
	parent context.Context
	kvs    map[string]string
}

func (c *RuntimeValuesContext) Deadline() (deadline time.Time, ok bool) {
	return c.parent.Deadline()
}

func (c *RuntimeValuesContext) Done() <-chan struct{} {
	return c.parent.Done()
}

func (c *RuntimeValuesContext) Err() error {
	return c.parent.Err()
}

func (c *RuntimeValuesContext) Value(key interface{}) interface{} {
	if kk, ok := key.(string); ok {
		for k, v := range c.kvs {
			if kk == k {
				return v
			}
		}
	}
	return c.parent.Value(key)
}
