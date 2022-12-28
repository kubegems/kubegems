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

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-envparse"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/edge/common"
	"kubegems.io/kubegems/pkg/edge/options"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/pprof"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const ClientIDSecret = "kubegems-edge-agent-id"

func Run(ctx context.Context, opts *options.AgentOptions) error {
	return run(ctx, opts)
}

type EdgeAgent struct {
	config       *rest.Config
	manufectures map[string]string
	clientID     string
	cluster      cluster.Interface
	tunserver    tunnel.GrpcTunnelServer
	httpapi      *AgentAPI
	options      *options.AgentOptions
	annotations  tunnel.Annotations
}

func run(ctx context.Context, options *options.AgentOptions) error {
	ctx = log.NewContext(ctx, log.LogrLogger)

	rest, err := kube.AutoClientConfig()
	if err != nil {
		return err
	}
	c, err := cluster.NewClusterAndStartWithIndexer(ctx, rest)
	if err != nil {
		return err
	}
	clientid, err := getClientID(ctx, c.GetClient(), options)
	if err != nil {
		return err
	}
	if clientid == "" {
		return fmt.Errorf("empty client id specified")
	}
	manufectures, err := ReadManufacture(options, clientid)
	if err != nil {
		return err
	}
	ea := &EdgeAgent{
		config:       rest,
		manufectures: manufectures,
		clientID:     clientid,
		options:      options,
		annotations:  nil,
		cluster:      c,
		httpapi:      &AgentAPI{cluster: c},
		tunserver:    tunnel.GrpcTunnelServer{TunnelServer: tunnel.NewTunnelServer(clientid, nil)},
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return ea.tunserver.ConnectUpstreamWithRetry(ctx, options.EdgeHubAddr, nil, "", ea.getAnnotations(ctx))
	})
	eg.Go(func() error {
		return ea.RunKeepAliveRouter(ctx, ea.options.KeepAliveInterval, ea.getAnnotations)
	})
	eg.Go(func() error {
		return ea.httpapi.Run(ctx, options.Listen)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}

func (ea *EdgeAgent) RunKeepAliveRouter(ctx context.Context, duration time.Duration, annotationsfunc func(ctx context.Context) tunnel.Annotations) error {
	log.Info("starting refresh router")

	if duration <= 0 {
		duration = 30 * time.Second
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			timer.Reset(duration)
			annotations := annotationsfunc(ctx)
			ea.tunserver.TunnelServer.SendKeepAlive(ctx, annotations)
		}
	}
}

func (ea *EdgeAgent) getAnnotations(ctx context.Context) tunnel.Annotations {
	if ea.annotations != nil {
		return ea.annotations
	}
	sv, _ := ea.cluster.Discovery().ServerVersion()
	nodeList, _ := ea.cluster.Kubernetes().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	annotations := map[string]string{
		common.AnnotationKeyEdgeAgentAddress:           "http://127.0.0.1" + ea.options.Listen,
		common.AnnotationKeyEdgeAgentRegisterAddress:   ea.options.EdgeHubAddr,
		common.AnnotationKeyEdgeAgentKeepaliveInterval: ea.options.KeepAliveInterval.String(),
		common.AnnotationKeyAPIserverAddress:           ea.config.Host,
		common.AnnotationKeyKubernetesVersion:          sv.String(),
		common.AnnotationKeyNodesCount:                 strconv.Itoa(len(nodeList.Items)),
	}
	maps.Copy(annotations, ea.manufectures)
	ea.annotations = annotations
	return annotations
}

const clientIDKey = "client-id"

const two = 2

func getClientID(ctx context.Context, cli client.Client, options *options.AgentOptions) (string, error) {
	clientid := ""
	// try secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClientIDSecret,
			Namespace: kube.LocalNamespaceOrDefault("kubegems-edge"),
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx, cli, secret, func() error {
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		if existsclientid := string(secret.Data[clientIDKey]); existsclientid == "" {
			clientid = uuid.NewString()
			secret.Data[clientIDKey] = []byte(clientid)
		} else {
			clientid = existsclientid
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return clientid, nil
}

func ReadManufacture(options *options.AgentOptions, clientID string) (map[string]string, error) {
	fullkvs := map[string]string{}
	for _, file := range options.ManufactureFile {
		kvs, err := ParseKV(file)
		if err != nil {
			return nil, err
		}
		maps.Copy(fullkvs, kvs)
	}

	// kvs from flag
	maps.Copy(fullkvs, ParseToMaps(options.Manufacture))

	// remap
	remapkeys := ParseToMaps(options.ManufactureRemap)
	for k, v := range remapkeys {
		val, ok := fullkvs[v]
		if ok {
			fullkvs[k] = val
		}
	}

	deviceid := options.DeviceID
	if deviceid == "" {
		// remap device id
		deviceid = fullkvs[options.DeviceIDKey]
	}
	if deviceid == "" {
		// default device id
		deviceid = clientID
	}
	fullkvs[common.AnnotationKeyDeviceID] = deviceid
	return fullkvs, nil
}

func ParseToMaps(list []string) map[string]string {
	// remap
	ret := map[string]string{}
	for _, item := range list {
		for _, kvstr := range strings.Split(item, ",") {
			splits := strings.SplitN(kvstr, "=", two)
			if len(splits) == two {
				key, value := splits[0], splits[1]
				ret[key] = value
			}
		}
	}
	return ret
}

func ParseKV(file string) (map[string]string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	kvs, err := ParseJSONFile(content)
	if err != nil {
		kvs, err = ParseKVFile(content)
		if err != nil {
			return nil, err
		}
	}
	return kvs, nil
}

// ParseKVFile parse kv from FOO="bar" likes file
func ParseKVFile(content []byte) (map[string]string, error) {
	return envparse.Parse(bytes.NewReader(content))
}

func ParseJSONFile(content []byte) (map[string]string, error) {
	kv := map[string]any{}

	d := json.NewDecoder(bytes.NewReader(content))
	d.UseNumber()
	if err := d.Decode(&kv); err != nil {
		return nil, err
	}
	ret := map[string]string{}
	for k, v := range kv {
		switch val := v.(type) {
		case string:
			ret[k] = val
		case bool:
			ret[k] = strconv.FormatBool(val)
		case json.Number:
			ret[k] = val.String()
		}
	}
	return ret, nil
}
