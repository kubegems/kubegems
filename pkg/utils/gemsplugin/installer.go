package gemsplugin

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	"kubegems.io/pkg/log"
)

type InstallerOptions struct {
	OperatorImage string  `json:"operator_image,omitempty"`
	InstallerYaml Plugins `json:"installer_yaml,omitempty"`
}

func DefaultInstallerOptions() *InstallerOptions {
	return &InstallerOptions{
		OperatorImage: "kubegems/installer-operator:v2.3-release",
		InstallerYaml: defaultInstallerObj,
	}
}

func (opts *InstallerOptions) JSON() []byte {
	bts, _ := json.Marshal(opts)
	return bts
}

func (opts *InstallerOptions) Name() string {
	return "Installer"
}

func (opts *InstallerOptions) Validate() error {
	return nil
}

var (
	defaultInstallerObj Plugins
)

func init() {
	if err := yaml.Unmarshal([]byte(defaultInstallerYaml), &defaultInstallerObj); err != nil {
		log.Fatalf("defaultInstallerYaml err: %v", err)
	}
}

const defaultInstallerYaml = `
apiVersion: plugins.kubegems.io/v1beta1
kind: Installer
metadata:
  name: kubegems-plugins
  namespace: kubegems-installer
spec:
  cluster_name: "{{ .Cluster.ClusterName }}"
  # Container runtime, this field used to record container's logs
  # runtime: docker or containerd
  runtime: "{{ .Cluster.Runtime }}"
  
  global:
    # Container repository on kubegems installer running.
    # default vvariable used to "docker.io/kubegems" if not set.
    # available container repositories "docker.io/kubegem" is default, other reigstry incoude <ghcr.io/kubegems> and <registry.cn-beijing.aliyuncs.com/kubegems>.
    # If you are using a private repository, you can configure a policy to replicat image locallly from any of the srouce registry listed above.
    repository: "{{ .Cluster.ImageRepo }}"
  
    # Secret for container repositories.
    # imagepullsecret: kubegems
  
    # Kubegems uses the built-in local-path-provisioner by defaults.
    # If you need to set a personalised storage class for the component, please configure it in the field "<component>.operator.persisten.storageclass".
    storageclass: "{{ .Cluster.DefaultStorageClass }}"
  
  core_plugins:
    kubegems_local:
      details:
        catalog: KubeGems
        description: KubeGems本地组件服务,运行在Kubernetes集群内部.
        version: "{{ .KubegemsVersion.GitVersion }}"
      namespace: gemcloud-system
      enabled: true
      operator:
        cert_manager:
          version: v1.4.0
          namespace: cert-manager
        basic_gateway:
          version: v1.0
          namespace: gemcloud-gateway-system
        gems_agent:
          # replicas: 1
        gems_controller:
          replicas: 1
      status:
        deployment:
        - gems-agent
        - gems-controller-manager
        required: true
  
    monitoring:
      details:
        catalog: 监控&告警
        description: KubeGems平台监控&告警控制器,包含Prometheus和AlertManager服务.
        version: v0.50.1-gems
      enabled: true
      namespace: gemcloud-monitoring-system
      operator:
        prometheus: 
          enabled: true
          replicas: 1
          retention: 30d
          apply_rules: true
          image:
            tag: 2.27.1-debian-10-r16
          resources:
            cpu: 4000m
            memory: 8Gi
          persistent:
            size: 50Gi
            # Specify stralgeclass to use, local-path was default value
            # storageclass: local-path
  
          # If you need to interface to an external alertmanager service, disealed alertmanager and configured the host field
          # Tips: host only support <ipaddress>:<ports>
          #external_host: 172.16.0.1:9093
        alertmanager:
          enabled: true
          replicas: 1
          image:
            tag: 0.22.2-debian-10-r2
          retention: 146h
          persistent:
            size: 10Gi
            # Specify stralgeclass to use, local-path was default value
            # storageclass: local-path
  
          # If you need to interface to an external alertmanager service, disealed alertmanager and configured the host field
          # Tips: host only support <ipaddress>:<ports>
          #external_host: 172.16.0.1:9093
      status:
        deployment:
        - prometheus-operator
  
    node_exporter:
      details:
        catalog: 监控&告警
        description: 物理机监控指标暴露器.
        version: v1.1.1-debian-10-r0
      enabled: true
      namespace: gemcloud-monitoring-system
      status:
        daemonset:
        - node-exporter
  
    kube_state_metrics:
      details:
        catalog: 监控&告警
        description: 监控Kubernetes内各个资源的运行状态.
        version: v1.9.8-debian-10-r0
      enabled: true
      namespace: gemcloud-monitoring-system
      status:
        deployment:
        - kube-state-metrics
  
    argo_rollouts:
      details:
        catalog: GitOps
        description: KubeGems内部应用策略部署的GitOps引擎,支持蓝绿、金丝雀发布等高级策略.
        version: v1.1.0
      enabled: true
      namespace: gemcloud-workflow-system
      status:
        deployment:
        - argo-rollouts
        required: true
  
    logging:
      details:
        catalog: 日志中心
        description: KubeGems平台管理容器日志框架,包含控制器、Loki Stack等服务.
        version: v3.15.0
      enabled: false
      namespace: gemcloud-logging-system
      operator:
        # Upstream used by logs whitch fluentbit collect, forward to fluentdpstream uspfluentbit and forwarded to flunetd
        enable_upstream: false
        fluentbit:
          # Set the buffer size for HTTP client when reading responses from Kubernetes API server. 
          # The value must be according to the Unit Size specification.
          #buffer: 256k
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
            cpu: "1"
            memory: 2Gi
          persistent:
            size: 10Gi
            #storageclass: local-path
        loki:
          enabled: true
          #image:
          #  repository: 
          #  tag:
          #resources:
          #  cpu: "4"
          #  memory: "8Gi"
          persistent:
            size: 100Gi
            #storageclass: local-path
          # If you need to interface to an external loki service, disealed loki and configured the host field
          # Tips: host only support <ipaddress>:<ports>
          #external_host: 172.168.0.1:3100
      status:
        deployment:
        - logging-operator
  
    eventer:
      details:
        catalog: 日志中心
        description: Kubernetes集群事件收集器(需开启 logging 套件).
        version: v1.1
      enabled: true
      namespace: gemcloud-logging-system
      status:
        deployment:
        - gems-eventer
  
    istio:
      details:
        catalog: 服务网格
        description: KubeGems平台服务治理套件.
        version: v1.11.7
      enabled: false
      namespace: istio-system
      operator:
        eastwestgateway:
          enabled: false
        dnsproxy:
          enabled: true
        istio-cni:
          enabled: true
        tracing:
          enabled: true
          param: 50
          address: "jaeger-collector.observability.svc.cluster.local:9411"
        kiali:
          enabled: true
          prometheus_urls: "http://prometheus.gemcloud-monitoring-system.svc.cluster.local:9090"
          trace_urls: "http://jaeger-query.observability.svc.cluster.local:16685/jaeger"
          grafana_urls: "http://grafana-service.gemcloud-monitoring-system.svc.cluster.local:3000"
      status:
        deployment:
        - istiod
  
    jaeger:
      details:
        catalog: 服务网格
        description: KubeGems平台服务追踪套件.
        version: v1.25.0
      enabled: false
      namespace: observability
      operator:
        sampling:
          type: probabilistic
          param: 0.5
        elasticsearch:
          enabled: true
          # Elasticsearch running mode, default is single node. <cluster> mode will be set 3 replicas as a cluster.
          mode: single
          persistent:
            size: 100Gi
            # storageclass: local-path
  
          # If you need to interface to an external ElasticSearch service, disealed ElasticSearch and configured the external_urls fielda.
          # external_urls: "http://172.16.0.1:9200"
      status:
        deployment:
        - jaeger-collector
        - jaeger-query
  
  kubernetes_plugins:
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
  
    node_problem_detector:
      details:
        catalog: 日志&事件
        description: Kubernees集群节点事件诊断器.
        version: v0.8.7
      enabled: false
      namespace: kube-system
      status:
        daemonset:
        - node-problem-detector
  
    node_local_dns:
      details:
        catalog: 网络
        description: Kuberntes主机DNS缓存服务.
        version: v1.15.13
      enabled: false
      operator:
        dns_upsteam: 192.168.0.10
      namespace: kube-system
      status:
        daemonset:
        - node-local-dns
  
    nvidia_device_plugin:
      details:
        catalog: 设备管理
        description: Nvidia公司为Kubernetes提供的云上容器独占显卡插件
        version: v1.0.0-beta
      enabled: false
      namespace: kube-system
      status:
        daemonset:
          - nvidia-device-plugin-daemonset
  
    gpu_manager:
      details:
        catalog: 设备管理
        description: 腾讯云(TKE)开源的GPU显卡资源虚拟化分配的Kubernetes插件.
        version: v1.1.2
      enabled: false
      namespace: kube-system
      status:
        daemonset:
        - tke-gpu-manager
`
