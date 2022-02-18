package clusterhandler

import "text/template"

var pluginsTpl = template.Must(template.New("plugins").Parse(`
apiVersion: plugins.kubegems.io/v1alpha1
kind: Installer
metadata:
  name: kubegems-plugins
  namespace: kubegems-installer
spec:
  cluster_name: {{ .Cluster.ClusterName }}
  # Container runtime, this field used to record container's logs
  # runtime: docker or containerd
  runtime: {{ .Cluster.Runtime }}
  core_plugins:
    alertmanager:
      details:
        catalog: 监控&告警
        description: 告警消息控制器，丰富的告警通知渠道，支持消息去重，降噪，分组，策略路由.
        version: 0.22.2-debian-10-r2
      enabled: true
      namespace: gemcloud-monitoring-system
      operator:
        alertmanager: gemcloud
        image:
          repository: harbor.kubegems.io/library/alertmanager
        persistent:
          size: 10Gi
          # Specify stralgeclass to use, local-path was default value
          #storageclass: local-path
        replicas: 3
        retention: 146h
      status:
        required: true
        statefulset:
        - alertmanager-gemcloud
      # If you need to interface to an external alertmanager service, disealed alertmanager and configured the host field
      # Tips: host only support <ipaddress>:<ports>
      # host: 172.16.0.1:9093
    argo_rollouts:
      details:
        catalog: DevOps
        description: Argo Rollout是一个运行在Kubernetes上的渐进式交付控制器.
        version: v1.1.0
      enabled: true
      namespace: gemcloud-workflow-system
      status:
        deployment:
        - argo-rollouts
        required: true
    basic_gateway:
      details:
        catalog: GemsCloud
        description: GemsCloud多租户网关控制器（基础版）
        version: v1.0
      enabled: true
      namespace: gemcloud-gateway-system
      status:
        deployment:
        - nginx-ingress-operator-controller-manager
        required: true
    cert_manager:
      details:
        catalog: GemsCloud
        description: Kubernetes上的全能证书管理工具.
        version: v1.4.0
      enabled: true
      namespace: cert-manager
      status:
        deployment:
        - cert-manager
        - cert-manager-cainjector
        - cert-manager-webhook
        required: true
    gems_agent:
      details:
        catalog: GemsCloud
        description: GemsCloud的集群客户端服务.
        version: {{ .Version.GitVersion }}
      enabled: true
      manual:
        image:
          repository: kubegems/kubegems
      namespace: gemcloud-system
      status:
        deployment:
        - gems-agent
        required: true
    gems_controller:
      details:
        catalog: GemsCloud
        description: GemsCloud的集群资源控制器.
        version: v0.3.6
      enabled: true
      manual:
        image:
          repository: kubegems/kubegems
      namespace: gemcloud-system
      status:
        deployment:
        - gems-controller-manager
        required: true
    logging:
      details:
        catalog: 日志中心
        description: 一个基于 kubenretes 的容器日志采集控制器.
        version: v3.15.0
      enabled: true
      operator:
        # Upstream used by logs whitch fluentbit collect, forward to fluentdpstream uspfluentbit and forwarded to flunetd
        enable_upstream: false
        fluentbit:
          resources:
            cpu: "2"
            memory: 1Gi
          # If the container logs are redirected to another path(not /var/log/pods), the path needs to be mounted to fluentbit.
          #volume_mounts:
          #  source: /data
          #  destination: /data
        fluentd:
          # The replicas of flunetd
          replicas: 2
          resources:
            cpu: "2"
            memory: 4Gi
          persistent:
            size: 100Gi
      namespace: gemcloud-logging-system
      status:
        deployment:
        - logging-operator
    loki:
      details:
        catalog: 日志中心
        description: 一个水平可扩展，高可用性，多租户的日志聚合系统.
        version: v2.4.1
      enabled: true
      manual:
        persistent:
          size: 500Gi
          storageclass: local-path
      namespace: gemcloud-logging-system
      status:
      # If you need to interface to an external loki service, disealed loki and configured the host field
      # Tips: host only support <ipaddress>:<ports>
      # host: 172.168.0.1:3100
        statefulset:
        - loki-system
    prometheus:
      details:
        catalog: 监控&告警
        description: Prometheus是一套开源的监控&告警框架.
        version: 2.27.1-debian-10-r16
      enabled: true
      namespace: gemcloud-monitoring-system
      operator:
        apply_rules: true
        image:
          repository: harbor.kubegems.io/library/prometheus
        resources:
          cpu: 4000m
          memory: 16Gi
        persistent:
          size: 400Gi
          #storageclass: local-path
        prometheus: gemcloud
        replicas: 1
        retention: 30d
      status:
        required: true
        statefulset:
        - prometheus-gemcloud
      # If you need to interface to an external prometheus service, disealed prometheus and configured the host field
      # Tips: host only support <ipaddress>:<ports>
      # host: 172.16.0.1:9090
  kubernetes_plugins:
    eventer:
      details:
        catalog: 日志&事件
        description: Kubernetes集群内事件收集器.
        version: v1.1
      enabled: true
      manual:
        image:
          repository: kubegems/kubegems
      namespace: gemcloud-logging-system
      status:
        deployment:
        - gems-eventer
    kube_state_metrics:
      details:
        catalog: 监控&告警
        description: 监控Kubernetes内各个资源的运行状态.
        version: 1.9.8-debian-10-r0
      enabled: true
      manual:
        image:
          repository: harbor.kubegems.io/library/kube-state-metrics
      namespace: kube-system
      status:
        deployment:
        - kube-state-metrics
        required: true
    local_path:
      details:
        catalog: 存储
        description: Rancher的开源的一个轻量级的卷管理控制器.
        version: v0.0.19
      enabled: true
      # Set kubernetes default StorageClass,if cluster don't have any storageclass.
      default_class: false
      namespace: local-path-storage
      status:
        deployment:
        - local-path-provisioner
        required: true
    metrics_server:
      details:
        catalog: 监控&告警
        description: Kubernetes内CPU和内存实时使用率，提供弹性伸缩容的资源判断.
        version: v0.4.2
      enabled: true
      manual:
        image:
          repository: harbor.kubegems.io/library/metrics-server
      namespace: kube-system
      status:
        deployment:
        - metrics-server
        required: true
    node_exporter:
      details:
        catalog: 监控&告警
        description: 集群内主机的详细监控客户端.
        version: 1.1.1-debian-10-r0
      enabled: true
      manual:
        image:
          repository: harbor.kubegems.io/library/node-exporter
      namespace: gemcloud-monitoring-system
      status:
        daemonset:
        - node-exporter
    node_problem_detector:
      details:
        catalog: 日志&事件
        description: 增强版的集群节点异常事件监控组件.
        version: v0.8.7
      enabled: false
      manual:
        image:
          repository: harbor.kubegems.io/library/node-problem-detector
      namespace: kube-system
      status:
        daemonset:
        - node-problem-detector
`))
