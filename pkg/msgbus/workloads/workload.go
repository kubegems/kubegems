package workloads

import (
	"context"
	"io/ioutil"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/msgbus/switcher"
	"github.com/kubegems/gems/pkg/utils/agents"
	"github.com/kubegems/gems/pkg/utils/msgbus"
)

type AgentMessageCollector struct {
	context       context.Context
	clientsSet    *agents.ClientSet
	ms            *switcher.MessageSwitcher
	clusterStatus sync.Map
	messageCh     chan *msgbus.NotifyMessage
}

func RunWorkloadCollector(ctx context.Context, cs *agents.ClientSet, ms *switcher.MessageSwitcher) error {
	ctrl := NewCtrl(ctx, cs, ms)
	return ctrl.Run()
}

func NewCtrl(ctx context.Context, agent *agents.ClientSet, ms *switcher.MessageSwitcher) *AgentMessageCollector {
	return &AgentMessageCollector{
		context:       ctx,
		clientsSet:    agent,
		ms:            ms,
		clusterStatus: sync.Map{},
		messageCh:     make(chan *msgbus.NotifyMessage, 1000),
	}
}

func (c *AgentMessageCollector) Refresh() {
	clusters := c.clientsSet.Clusters()
	clusterMap := map[string]interface{}{}
	for _, cluster := range clusters {
		clusterMap[cluster] = cluster
		if _, ok := c.clusterStatus.Load(cluster); !ok {
			stopch := make(chan struct{})
			go c.MessageChan(c.context, cluster, stopch)
			c.clusterStatus.Store(cluster, stopch)
		}
	}
	var toStopCluster []string
	c.clusterStatus.Range(func(k, v interface{}) bool {
		if _, exist := clusterMap[k.(string)]; !exist {
			toStopCluster = append(toStopCluster, k.(string))
		}
		return true
	})
	for _, clusterName := range toStopCluster {
		ch, exist := c.clusterStatus.Load(clusterName)
		if !exist {
			continue
		}
		stopch := ch.(chan struct{})
		stopch <- struct{}{}
		c.clusterStatus.Delete(clusterName)
	}
}

func (c *AgentMessageCollector) Run() error {
	clusters := c.clientsSet.Clusters()
	msgswitch := c.ms
	for _, cluster := range clusters {
		stopch := make(chan struct{})
		go c.MessageChan(c.context, cluster, stopch)
		c.clusterStatus.Store(cluster, stopch)
	}

	for {
		select {
		case <-c.context.Done():
			return c.context.Err()
		case msg := <-c.messageCh:
			msgswitch.DispatchMessage(msg)
		}
	}
}

func (c *AgentMessageCollector) MessageChan(ctx context.Context, clustername string, stopch chan struct{}) {
	uri := "/notify"
	for {
		func() error {
			clusterProxy, err := c.clientsSet.ClientOf(ctx, clustername)
			if err != nil {
				log.WithField("cluster", clustername).Warnf("get proxy failed: %v", err)
				return err
			}
			url := getUrl(clusterProxy.BaseAddr, uri)
			conn, resp, err := clusterProxy.ProxyClient.WebsockerDialer.DialContext(ctx, url, nil)
			if err != nil {
				content := ""
				if resp != nil {
					t, _ := ioutil.ReadAll(resp.Body)
					content = string(t)
				}
				log.WithField("cluster", clustername).Warnf("connect failed: %s %v %s", url, err, content)
				return err
			}
			for {
				tmp := msgbus.NotifyMessage{}
				if err := conn.ReadJSON(&tmp); err != nil {
					conn.Close()
					log.WithField("cluster", clustername).Warnf("read failed: %s %v", url, err)
					return err
				}
				switch tmp.MessageType {
				case msgbus.Changed:
					tmp.InvolvedObject.Cluster = clustername
					c.messageCh <- &tmp
				case msgbus.Alert:
					c.messageCh <- &tmp
				}
			}
		}()
		// 删除逻辑，等到出错后再处理
		select {
		case <-stopch:
			return
		default:
			time.Sleep(time.Second * 5)
		}
	}
}

func getUrl(baseaddr *url.URL, uri string) string {
	var scheme string
	if baseaddr.Scheme == "http" {
		scheme = "ws"
	} else {
		scheme = "wss"
	}
	wsu := &url.URL{
		Scheme: scheme,
		Host:   baseaddr.Host,
		Path:   path.Join(baseaddr.Path + uri),
	}
	return wsu.String()
}
