# monitoring

## kube-prometheus-stack

kube-prometheus-stack 包含了 prometheus 和 grafana 以及其周边工具，例如 kube-state-metrics,node-exporter 等。

## grafana

grafana helm chart 增加了通过 sidecar watch k8s resource 进行动态更新 grafana 配置功能,其通过[kiwigrid/k8s-sidecar](https://github.com/kiwigrid/k8s-sidecar)实现。

该功能默认关闭，需要以下配置开启：

```yaml
grafana:
  sidecar:
  dashboards:
    enabled: true
  datasources:
    enabled: true
```

通过 configmap 配置 dashboard :
