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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/printers"
	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/kube"
)

const (
	AnnotationKeyEdgeHubAddress = "edge.kubegems.io/edge-hub-address"
	AnnotationKeyEdgeHubCert    = "edge.kubegems.io/edge-hub-key"
	AnnotationKeyEdgeHubCA      = "edge.kubegems.io/edge-hub-ca"
	AnnotationKeyEdgeHubKey     = "edge.kubegems.io/edge-hub-cert"
	LabelKeIsyEdgeHub           = "edge.kubegems.io/is-edge-hub"

	AnnotationKeyEdgeAgentAddress         = "edge.kubegems.io/edge-agent-address"
	AnnotationKeyEdgeAgentRegisterAddress = "edge.kubegems.io/edge-agent-register-address"
	AnnotationKeyKubernetesVersion        = "edge.kubegems.io/kubernetes-version"
	AnnotationKeyAPIserverAddress         = "edge.kubegems.io/apiserver-address"

	// temporary connection do not write to database
	AnnotationIsTemporaryConnect = "edge.kubegems.io/temporary-connect"
)

type EdgeClusterManager struct {
	SelfAddress string
	Store       EdgeClusterStore
}

func NewClusterManager(store EdgeClusterStore, selfhost string) *EdgeClusterManager {
	return &EdgeClusterManager{
		Store:       store,
		SelfAddress: selfhost,
	}
}

func (m *EdgeClusterManager) List(ctx context.Context, labels labels.Selector) ([]v1beta1.EdgeCluster, error) {
	_, list, err := m.Store.List(ctx, ListOptions{Selector: labels})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (m *EdgeClusterManager) ListPage(ctx context.Context, page, size int, labels labels.Selector) (response.TypedPage[v1beta1.EdgeCluster], error) {
	total, list, err := m.Store.List(ctx, ListOptions{Page: page, Size: size, Selector: labels})
	if err != nil {
		return response.TypedPage[v1beta1.EdgeCluster]{}, err
	}
	return response.TypedPage[v1beta1.EdgeCluster]{
		Total:       int64(total),
		List:        list,
		CurrentPage: int64(page),
		CurrentSize: int64(size),
	}, nil
}

func (m *EdgeClusterManager) Get(ctx context.Context, uid string) (*v1beta1.EdgeCluster, error) {
	return m.Store.Get(ctx, uid)
}

func (m *EdgeClusterManager) Delete(ctx context.Context, uid string) (*v1beta1.EdgeCluster, error) {
	return m.Store.Delete(ctx, uid)
}

func (m *EdgeClusterManager) Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeCluster) error) (*v1beta1.EdgeCluster, error) {
	return m.Store.Update(ctx, name, fun)
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
func (m *EdgeClusterManager) PreCreate(ctx context.Context, example *v1beta1.EdgeCluster) (*v1beta1.EdgeCluster, error) {
	// check hub is already exists
	_, err := m.Store.Get(ctx, example.Spec.Register.HubName)
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
		if in.Status.Phase != v1beta1.EdgeClusterPhaseOnline {
			in.Status.Phase = v1beta1.EdgeClusterPhaseWaiting
		}
		selfaddr := m.SelfAddress
		if !strings.HasPrefix(selfaddr, "http") {
			selfaddr = "http://" + selfaddr
		}
		manifestAddress := fmt.Sprintf("%s/v1/edge-clusters/%s/agent-installer.yaml?token=%s", selfaddr, in.Name, in.Spec.Register.BootstrapToken)
		in.Status.Register.URL = manifestAddress
		return nil
	}
	return m.Store.Update(ctx, example.Name, updatespec)
}

type InstallerTemplateValues struct {
	EdgeAddress string
	AgentImage  string
	TLSCert     []byte
	TLSKey      []byte
	TLSCA       []byte
}

func (m *EdgeClusterManager) RenderInstallManifests(ctx context.Context, uid, token string) ([]byte, error) {
	exists, err := m.Store.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	if exists.Spec.Register.BootstrapToken != token {
		return nil, fmt.Errorf("invalid token: %s", token)
	}
	if exists.Spec.Register.HubName == "" {
		return nil, fmt.Errorf("no hub name specified for the edge cluster")
	}
	hub, err := m.Store.Get(ctx, exists.Spec.Register.HubName)
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
	if _, err := m.Store.Update(ctx, uid, func(cluster *v1beta1.EdgeCluster) error {
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

func (m *EdgeClusterManager) gencert(cn string, expire *time.Time, hub *v1beta1.EdgeCluster) (*v1beta1.Certs, error) {
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

func (m *EdgeClusterManager) SetTunnelConnectedStatus(ctx context.Context, name string, connected bool, anno map[string]string) error {
	if istemp, _ := strconv.ParseBool(anno[AnnotationIsTemporaryConnect]); istemp {
		log.Info("ignore temporary connection", "name", name, "annotations", anno)
		return nil
	}
	updatefunc := func(cluster *v1beta1.EdgeCluster) error {
		now := metav1.Now()
		if connected {
			if _, ok := anno[AnnotationKeyEdgeHubAddress]; ok {
				if cluster.Labels == nil {
					cluster.Labels = make(map[string]string)
				}
				cluster.Labels[LabelKeIsyEdgeHub] = "true"
			}
			cluster.Status.Tunnel.Connected = true
			cluster.Status.Tunnel.LastOnlineTimestamp = &now
			cluster.Status.Phase = v1beta1.EdgeClusterPhaseOnline
			cluster.Status.Manufacture = anno // annotations as manufacture set
		} else {
			cluster.Status.Tunnel.LastOfflineTimestamp = &now
			cluster.Status.Phase = v1beta1.EdgeClusterPhaseOffline
			cluster.Status.Tunnel.Connected = false
		}
		return nil
	}
	_, err := m.Store.Update(ctx, name, updatefunc)
	return err
}

func (s *EdgeClusterManager) SyncTunnelStatusFrom(ctx context.Context, server *tunnel.TunnelServer) error {
	logr.FromContextOrDiscard(ctx).Info("start syncing tunnel status")
	watcher := server.Wacth(ctx)
	defer watcher.Close()

	for event := range watcher.Result() {
		switch event.Kind {
		case tunnel.EventKindConnected:
			for id, anno := range event.Peers {
				if err := s.SetTunnelConnectedStatus(ctx, id, true, anno); err != nil {
					log.Error(err, "set to online", "id", id)
				}
			}
		case tunnel.EventKindDisConnected:
			for id := range event.Peers {
				if err := s.SetTunnelConnectedStatus(ctx, id, false, nil); err != nil {
					log.Error(err, "set to offline", "id", id)
				}
			}
		default:
			log.Info("invalid event exit watcher")
			break
		}
	}
	return nil
}
