package kubeclient

import (
	"context"
	"net/http"
	"time"

	"kubegems.io/pkg/log"
)

func (k KubeClient) IsClusterHealth(cluster string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errch := make(chan error)
	go func(_ context.Context, ch chan error) {
		ch <- k.DoRequest(http.MethodGet, cluster, "/healthz", nil, nil)
	}(ctx, errch)

	select {
	case <-ctx.Done(): // 超时
		log.Warnf("request to cluster %s timeout", cluster)
		return false
	case err := <-errch:
		return err == nil
	}
}
