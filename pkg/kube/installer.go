package kube

import (
	"bytes"
	"context"
	"text/template"

	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/utils"
	"github.com/kubegems/gems/pkg/version"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	installerNamespace = "kubegems-installer"
)

type InstallerOptions struct {
	Version string `yaml:"version"`
}

func DefaultInstallerOptions() *InstallerOptions {
	return &InstallerOptions{
		Version: "latest",
	}
}

func (i *InstallerOptions) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&i.Version, utils.JoinFlagName(prefix, "version"), "latest", "installer version")
}

type InstallFields struct {
	*models.Cluster
	version.Version
	*InstallerOptions
}

func (i *ClusterInstaller) Install() error {
	if err := i.CreateNamespaceIfNotExists(); err != nil {
		return err
	}

	fields := InstallFields{
		Cluster:          i.Cluster,
		Version:          version.Get(),
		InstallerOptions: i.InstallerOptions,
	}

	// install crd
	t1, err := template.New("installer").Parse(installYAML)
	if err != nil {
		log.Error(err, "parse installer yaml template")
		return err
	}
	installerbuf := new(bytes.Buffer)
	if err := t1.Execute(installerbuf, fields); err != nil {
		log.Error(err, "execute installer yaml template")
		return err
	}
	if err := CreateByYamlOrJson(context.TODO(), i.config, installerbuf.Bytes()); err != nil {
		log.Error(err, "create installer yaml")
		return err
	}

	// install plugin, 与crd分开部署，以刷新restmap
	t2, err := template.New("plugins").Parse(pluginYAML)
	if err != nil {
		log.Error(err, "parse plugins yaml template")
		return err
	}
	pluginsbuf := new(bytes.Buffer)
	if err := t2.Execute(pluginsbuf, fields); err != nil {
		log.Error(err, "execute plugins yaml template")
		return err
	}
	return CreateByYamlOrJson(context.TODO(), i.config, pluginsbuf.Bytes())
}

func (i *ClusterInstaller) Uninstall() error {
	fields := InstallFields{
		Cluster:          i.Cluster,
		Version:          version.Get(),
		InstallerOptions: i.InstallerOptions,
	}

	// uninstall crd
	t1, err := template.New("installer").Parse(installYAML)
	if err != nil {
		log.Error(err, "parse installer yaml template")
		return err
	}
	installerbuf := new(bytes.Buffer)
	if err := t1.Execute(installerbuf, fields); err != nil {
		log.Error(err, "execute installer yaml template")
		return err
	}
	if err := DeleteByYamlOrJson(context.TODO(), i.config, installerbuf.Bytes()); err != nil {
		log.Error(err, "delete installer yaml")
		return err
	}

	t2, err := template.New("plugins").Parse(pluginYAML)
	if err != nil {
		log.Error(err, "parse plugins yaml template")
		return err
	}
	pluginsbuf := new(bytes.Buffer)
	if err := t2.Execute(pluginsbuf, fields); err != nil {
		log.Error(err, "execute plugins yaml template")
		return err
	}
	return DeleteByYamlOrJson(context.TODO(), i.config, pluginsbuf.Bytes())
}

type ClusterInstaller struct {
	*models.Cluster
	client *kubernetes.Clientset
	config *rest.Config
	*InstallerOptions
}

func NewClusterInstaller(config *rest.Config, client *kubernetes.Clientset,
	cluster *models.Cluster, opts *InstallerOptions) *ClusterInstaller {
	return &ClusterInstaller{
		Cluster:          cluster,
		client:           client,
		config:           config,
		InstallerOptions: opts,
	}
}

func (i *ClusterInstaller) CreateNamespaceIfNotExists() error {
	_, err := i.client.CoreV1().Namespaces().Get(context.Background(), installerNamespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if errors.IsNotFound(err) {
		if _, err = i.client.CoreV1().Namespaces().Create(context.Background(), i.getNamespceObj(), metav1.CreateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (i *ClusterInstaller) getNamespceObj() *corev1.Namespace {
	ns := &corev1.Namespace{}
	ns.Name = installerNamespace
	return ns
}

const installYAML = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: installers.plugins.gems.cloudminds.com
spec:
  group: plugins.gems.cloudminds.com
  names:
    kind: Installer
    listKind: InstallerList
    plural: installers
    singular: installer
  scope: Namespaced
  versions:
    - name: v1alpha1
      schema:
        openAPIV3Schema:
          description: Installer is the Schema for the installers API
          properties:
            apiVersion:
              description:
                "APIVersion defines the versioned schema of this representation
                of an object. Servers should convert recognized schemas to the latest
                internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources"
              type: string
            kind:
              description:
                "Kind is a string value representing the REST resource this
                object represents. Servers may infer this from the endpoint the client
                submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds"
              type: string
            metadata:
              type: object
            spec:
              description: Spec defines the desired state of Installer
              type: object
              x-kubernetes-preserve-unknown-fields: true
            status:
              description: Status defines the observed state of Installer
              type: object
              x-kubernetes-preserve-unknown-fields: true
          type: object
      served: true
      storage: true
      subresources:
        status: {}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubegems-installer
  namespace: kubegems-installer
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kubegems-installer-leader-election-role
  namespace: kubegems-installer
rules:
  - apiGroups:
      - ""
    resources:
      - configmaps
    verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
  - apiGroups:
      - coordination.k8s.io
    resources:
      - leases
    verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
  - apiGroups:
      - ""
    resources:
      - events
    verbs:
      - create
      - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubegems-installer-role
rules:
  - apiGroups:
      - ""
    resources:
      - secrets
      - pods
      - pods/exec
      - pods/log
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
  - apiGroups:
      - apps
    resources:
      - deployments
      - daemonsets
      - replicasets
      - statefulsets
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
  - apiGroups:
      - plugins.gems.cloudminds.com
    resources:
      - installers
      - installers/status
      - installers/finalizers
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubegems-installer-metrics-reader
rules:
  - nonResourceURLs:
      - /metrics
    verbs:
      - get
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubegems-installer-proxy-role
rules:
  - apiGroups:
      - authentication.k8s.io
    resources:
      - tokenreviews
    verbs:
      - create
  - apiGroups:
      - authorization.k8s.io
    resources:
      - subjectaccessreviews
    verbs:
      - create
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kubegems-installer-leader-election-rolebinding
  namespace: kubegems-installer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: gems-installer-leader-election-role
subjects:
  - kind: ServiceAccount
    name: kubegems-installer
    namespace: kubegems-installer
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubegems-installer-manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: kubegems-installer
    namespace: kubegems-installer
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubegems-installer-proxy-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kubegems-installer-proxy-role
subjects:
  - kind: ServiceAccount
    name: kubegems-installer
    namespace: kubegems-installer
---
apiVersion: v1
data:
  controller_manager_config.yaml: |
    apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
    kind: ControllerManagerConfig
    health:
      healthProbeBindAddress: :6789
    metrics:
      bindAddress: 127.0.0.1:8080
    leaderElection:
      leaderElect: true
      resourceName: 811c9dc5.gems.cloudminds.com
kind: ConfigMap
metadata:
  name: kubegems-installer-manager-config
  namespace: kubegems-installer
---
apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: kubegems-installer-manager
  name: kubegems-installer-metrics
  namespace: kubegems-installer
spec:
  ports:
    - name: https
      port: 8443
      protocol: TCP
      targetPort: https
  selector:
    control-plane: controller-manager
    app.kubernetes.io/name: kubegems-installer-manager
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: kubegems-installer-manager
    control-plane: controller-manager
  name: kubegems-installer-manager
  namespace: kubegems-installer
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: kubegems-installer-manager
      control-plane: controller-manager
  strategy:
    rollingUpdate:
    type: Recreate
  template:
    metadata:
      labels:
        app.kubernetes.io/name: kubegems-installer-manager
        control-plane: controller-manager
    spec:
      containers:
      - args:
        - secure-listen-address=0.0.0.0:8443
        - upstream=http://127.0.0.1:8081/
        - logtostderr=true
        - v=10
        image: harbor.cloudminds.com/library/kube-rbac-proxy:v0.8.0
        imagePullPolicy: IfNotPresent
        name: kube-rbac-proxy
        ports:
        - containerPort: 8443
          name: https
          protocol: TCP
      - args:
        - health-probe-bind-address=:6789
        - metrics-bind-address=127.0.0.1:8081
        - leader-elect
        - leader-election-id=plugins
        env:
        - name: ANSIBLE_GATHERING
          value: explicit
        - name: ENABLED_KUBEGEMS_DOCS
          value: "off"
        image: harbor.cloudminds.com/library/installer-operator:{{ .InstallerOptions.Version }}
        imagePullPolicy: Always
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /healthz
            port: 6789
            scheme: HTTP
          initialDelaySeconds: 15
          periodSeconds: 20
          successThreshold: 1
          timeoutSeconds: 1
        name: manager
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /readyz
            port: 6789
            scheme: HTTP
          initialDelaySeconds: 5
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        securityContext:
          allowPrivilegeEscalation: false
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext:
        runAsNonRoot: true
      serviceAccount: kubegems-installer
      serviceAccountName: kubegems-installer
`

const pluginYAML = `
apiVersion: plugins.gems.cloudminds.com/v1alpha1
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
          repository: harbor.cloudminds.com/library/alertmanager
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
          repository: harbor.cloudminds.com/gemscloud/gems-agent
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
          repository: harbor.cloudminds.com/gemscloud/gems-controller
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
          repository: harbor.cloudminds.com/library/prometheus
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
          repository: harbor.cloudminds.com/gemscloud/gems-eventer
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
          repository: harbor.cloudminds.com/library/kube-state-metrics
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
          repository: harbor.cloudminds.com/library/metrics-server
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
          repository: harbor.cloudminds.com/library/node-exporter
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
          repository: harbor.cloudminds.com/library/node-problem-detector
      namespace: kube-system
      status:
        daemonset:
        - node-problem-detector`
