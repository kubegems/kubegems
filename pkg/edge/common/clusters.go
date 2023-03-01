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

package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/printers"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgcluster "sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	AnnotationKeyEdgeHubAddress = "edge.kubegems.io/edge-hub-address"
	AnnotationKeyEdgeHubCert    = "edge.kubegems.io/edge-hub-key"
	AnnotationKeyEdgeHubCA      = "edge.kubegems.io/edge-hub-ca"
	AnnotationKeyEdgeHubKey     = "edge.kubegems.io/edge-hub-cert"
	LabelKeIsyEdgeHub           = "edge.kubegems.io/is-edge-hub"

	AnnotationKeyEdgeAgentAddress           = "edge.kubegems.io/edge-agent-address"
	AnnotationKeyEdgeAgentKeepaliveInterval = "edge.kubegems.io/edge-agent-keepalive-interval"
	AnnotationKeyEdgeAgentRegisterAddress   = "edge.kubegems.io/edge-agent-register-address"
	AnnotationKeyKubernetesVersion          = "edge.kubegems.io/kubernetes-version"
	AnnotationKeyAPIserverAddress           = "edge.kubegems.io/apiserver-address"
	AnnotationKeyNodesCount                 = "edge.kubegems.io/nodes-count"
	AnnotationKeyDeviceID                   = "edge.kubegems.io/device-id"

	// temporary connection do not write to database
	AnnotationIsTemporaryConnect = "edge.kubegems.io/temporary-connect"
)

type EdgeManager struct {
	SelfAddress  string
	ClusterStore EdgeClusterStore
	HubStore     EdgeHubStore
}

func NewClusterManager(ctx context.Context, namespace string, selfhost string) (*EdgeManager, error) {
	if namespace == "" {
		namespace = kube.LocalNamespaceOrDefault(gems.NamespaceEdge)
	}
	cfg, err := kube.AutoClientConfig()
	if err != nil {
		return nil, err
	}
	apply := func(c pkgcluster.Cluster) error {
		// add device id index
		return c.GetCache().IndexField(ctx, &v1beta1.EdgeCluster{}, "device-id", func(o client.Object) []string {
			cluster, ok := o.(*v1beta1.EdgeCluster)
			if !ok {
				return nil
			}
			return []string{cluster.Status.Manufacture[AnnotationKeyDeviceID]}
		})
	}
	c, err := cluster.NewClusterAndStart(ctx, cfg, apply, cluster.WithInNamespace(namespace))
	if err != nil {
		return nil, err
	}
	return &EdgeManager{
		ClusterStore: EdgeClusterK8sStore{cli: c.GetClient(), ns: namespace},
		HubStore:     EdgeHubK8sStore{cli: c.GetClient(), ns: namespace},
		SelfAddress:  selfhost,
	}, nil
}

func (m *EdgeManager) ListPage(
	ctx context.Context,
	opts request.ListOptions,
	labels, manufacture labels.Selector,
) (response.Page[v1beta1.EdgeCluster], error) {
	total, list, err := m.ClusterStore.List(ctx, ListOptions{
		Page:        opts.Page,
		Size:        opts.Size,
		Search:      opts.Search,
		Selector:    labels,
		Manufacture: manufacture,
	})
	if err != nil {
		return response.Page[v1beta1.EdgeCluster]{}, err
	}
	return response.Page[v1beta1.EdgeCluster]{
		Total: int64(total),
		List:  list,
		Page:  int64(opts.Page),
		Size:  int64(opts.Size),
	}, nil
}

type PrecreateOptions struct {
	UID          string            `json:"uid,omitempty"`
	HubName      string            `json:"hubName,omitempty"`      // hub name edge cluster registered to
	Annotations  map[string]string `json:"annotations,omitempty"`  // edge annotations
	Labels       map[string]string `json:"labels,omitempty"`       // edge labels
	AgentImage   string            `json:"agentImage,omitempty"`   // agent image edge cluster used to register
	CreateCert   bool              `json:"createCert,omitempty"`   // pre generated edge certificate
	CertExpireAt *time.Time        `json:"certExpireAt,omitempty"` // the expiration of the certificate
}

// return a register address
func (m *EdgeManager) PreCreate(ctx context.Context, example *v1beta1.EdgeCluster) (*v1beta1.EdgeCluster, error) {
	// check hub is already exists
	_, err := m.HubStore.Get(ctx, example.Spec.Register.HubName)
	if err != nil {
		return nil, fmt.Errorf("get edge hub %s: %w", example.Spec.Register.HubName, err)
	}
	updatespec := func(in *v1beta1.EdgeCluster) error {
		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		for k, v := range example.Annotations {
			in.Annotations[k] = v
		}
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		for k, v := range example.Labels {
			in.Labels[k] = v
		}
		in.Spec.Register = example.Spec.Register
		if in.Status.Phase != v1beta1.EdgePhaseOnline {
			in.Status.Phase = v1beta1.EdgePhaseWaiting
		}
		selfaddr := m.SelfAddress
		if !strings.HasPrefix(selfaddr, "http") {
			selfaddr = "http://" + selfaddr
		}
		manifestAddress := fmt.Sprintf("%s/v1/edge-clusters/%s/agent-installer.yaml?token=%s", selfaddr, in.Name, in.Spec.Register.BootstrapToken)
		in.Status.Register.URL = manifestAddress
		return nil
	}
	return m.ClusterStore.Update(ctx, example.Name, updatespec)
}

type InstallerTemplateValues struct {
	EdgeAddress string
	AgentImage  string
	TLSCert     []byte
	TLSKey      []byte
	TLSCA       []byte
}

func (m *EdgeManager) RenderInstallManifests(ctx context.Context, uid, token string) ([]byte, error) {
	exists, err := m.ClusterStore.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	if exists.Spec.Register.BootstrapToken != token {
		return nil, fmt.Errorf("invalid token: %s", token)
	}
	if exists.Spec.Register.HubName == "" {
		return nil, fmt.Errorf("no hub name specified for the edge cluster")
	}
	hub, err := m.HubStore.Get(ctx, exists.Spec.Register.HubName)
	if err != nil {
		return nil, err
	}
	hubaddress := hub.Status.Manufacture[AnnotationKeyEdgeHubAddress]
	if hubaddress == "" {
		return nil, fmt.Errorf("edge hub %s has no address", hub.Name)
	}
	edgecerts := exists.Spec.Register.Certs
	// use pre generated certificate
	if edgecerts == nil {
		log.Info("create edge certificate", "uid", uid)
		expire := (*time.Time)(nil)
		if presetExpr := exists.Spec.Register.ExpiresAt; presetExpr != nil {
			expire = &presetExpr.Time
		}
		generated, err := m.gencert(uid, expire, hub)
		if err != nil {
			return nil, err
		}
		edgecerts = generated
	}
	// update register status
	if _, err := m.ClusterStore.Update(ctx, uid, func(cluster *v1beta1.EdgeCluster) error {
		now := metav1.Now()
		cluster.Status.Register.LastRegister = &now
		cluster.Status.Register.LastRegisterToken = token
		return nil
	}); err != nil {
		return nil, err
	}
	// render template
	objects := RenderManifets(uid, exists.Spec.Register.Image, hubaddress, *edgecerts)
	printer := printers.YAMLPrinter{}
	buf := bytes.NewBuffer(nil)
	for _, obj := range objects {
		kube.FillGVK(obj, kube.GetScheme())
		printer.PrintObj(obj, buf)
	}
	return buf.Bytes(), nil
}

func (m *EdgeManager) gencert(cn string, expire *time.Time, hub *v1beta1.EdgeHub) (*v1beta1.Certs, error) {
	hubcert := v1beta1.Certs{
		CA:   []byte(hub.Status.Manufacture[AnnotationKeyEdgeHubCA]),
		Cert: []byte(hub.Status.Manufacture[AnnotationKeyEdgeHubCert]),
		Key:  []byte(hub.Status.Manufacture[AnnotationKeyEdgeHubKey]),
	}
	if len(hubcert.Cert) == 0 || len(hubcert.Key) == 0 {
		return nil, fmt.Errorf("edge hub %s dont have certificate in status manufacture", hub.Name)
	}
	certpem, keypem, err := SignCertificate(hubcert.CA, hubcert.Cert, hubcert.Key, CertOptions{
		CommonName: cn,
		ExpireAt:   expire,
	})
	if err != nil {
		return nil, err
	}
	edgecerts := &v1beta1.Certs{CA: hubcert.Cert, Cert: certpem, Key: keypem}
	return edgecerts, nil
}

func (m *EdgeManager) OnTunnelConnectedStatusChange(ctx context.Context,
	connected bool, isrefresh bool,
	fromname string, fromannotations map[string]string,
	name string, anno map[string]string,
) error {
	// is temporary connection
	if istemp, _ := strconv.ParseBool(anno[AnnotationIsTemporaryConnect]); istemp {
		log.Info("ignore temporary connection", "from", fromname, "name", name, "annotations", anno)
		return nil
	}
	now := metav1.Now()

	// is edge hub
	if address, ok := anno[AnnotationKeyEdgeHubAddress]; ok {
		// edgehub do not update heartbeat
		if isrefresh {
			return nil
		}
		log.Info("set hub tunnel status", "name", name, "connected", connected)
		_, err := m.HubStore.Update(ctx, name, func(cluster *v1beta1.EdgeHub) error {
			cluster.Status.Tunnel.Connected = connected
			cluster.Status.Manufacture = anno // annotations as manufacture set
			if connected {
				cluster.Status.Address = address
				cluster.Status.Tunnel.LastOnlineTimestamp = &now
				cluster.Status.Phase = v1beta1.EdgePhaseOnline
			} else {
				cluster.Status.Tunnel.LastOfflineTimestamp = &now
				cluster.Status.Phase = v1beta1.EdgePhaseOffline
			}
			return nil
		})
		return err
	}

	// is edge cluster
	_, err := m.ClusterStore.Update(ctx, name, func(cluster *v1beta1.EdgeCluster) error {
		cluster.Status.Tunnel.Connected = connected

		// set hub address from hub address
		if !isrefresh {
			if val := fromannotations[AnnotationKeyEdgeHubAddress]; val != "" {
				anno[AnnotationKeyEdgeAgentRegisterAddress] = val
			}
		}
		if deviceid := anno[AnnotationKeyDeviceID]; deviceid != "" {
			if cluster.Labels == nil {
				cluster.Labels = map[string]string{}
			}
			// set device id in label to select
			cluster.Labels[AnnotationKeyDeviceID] = deviceid
		}
		cluster.Status.Manufacture = anno // annotations as manufacture set
		if connected {
			if isrefresh {
				log.Info("set heartbeat status", "from", fromname, "name", name)
				cluster.Status.Tunnel.LastHeartBeatTimestamp = &now
			} else {
				log.Info("set tunnel status", "from", fromname, "name", name)
				cluster.Status.Tunnel.LastOnlineTimestamp = &now
			}
			cluster.Status.Phase = v1beta1.EdgePhaseOnline
		} else {
			cluster.Status.Tunnel.LastOfflineTimestamp = &now
			cluster.Status.Phase = v1beta1.EdgePhaseOffline
		}
		return nil
	})
	return err
}

func (s *EdgeManager) SyncTunnelStatusFrom(ctx context.Context, server *tunnel.TunnelServer) error {
	logr.FromContextOrDiscard(ctx).Info("start syncing tunnel status")
	watcher := server.Wacth(ctx)
	defer watcher.Close()

	for event := range watcher.Result() {
		for name, anno := range event.Peers {
			if err := s.OnTunnelConnectedStatusChange(ctx,
				event.Kind != tunnel.EventKindDisConnected, // is online
				event.Kind == tunnel.EventKindKeepalive,    // is online keepalive
				event.From, event.FromAnnotations,
				name, anno); err != nil {
				log.Error(err, "set to online", "id", name)
			}
		}
	}
	return errors.New("watcher exit")
}
