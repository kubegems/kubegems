package clusterhandler

import "text/template"

var installerOperatorTpl = template.Must(template.New("installer-operator").Parse(`
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: installers.plugins.kubegems.io
spec:
  group: plugins.kubegems.io
  names:
    kind: Installer
    listKind: InstallerList
    plural: installers
    singular: installer
  scope: Namespaced
  versions:
  - name: v1beta1
    schema:
      openAPIV3Schema:
        description: Installer is the Schema for the installers API
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
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
      - plugins.kubegems.io
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
      resourceName: 811c9dc5.kubegems.io
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
        image: kubegems/kube-rbac-proxy:v0.8.0
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
        - name: RUNNING_MODE 
          value: "worker"
        image: {{ .InstallerOptions.OperatorImage }}
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
`))
