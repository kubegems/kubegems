# kubegems-models

%%DESCRIPTION%% (check existing examples)

## TL;DR

```console
$ helm repo add kubegems https://charts.kubegems.io/kubegems
$ helm install my-release kubegems/kubegems-models
```

## Introduction

%%INTRODUCTION%% (check existing examples)

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- PV provisioner support in the underlying infrastructure
- ReadWriteMany volumes for deployment scaling

## Installing the Chart

To install the chart with the release name `my-release`:

```console
helm install my-release kubegems/kubegems-models
```

The command deploys kubegems-models on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

### Global parameters

| Name                      | Description                                     | Value |
| ------------------------- | ----------------------------------------------- | ----- |
| `global.imageRegistry`    | Global Docker image registry                    | `""`  |
| `global.imagePullSecrets` | Global Docker registry secret names as an array | `[]`  |
| `global.storageClass`     | Global StorageClass for Persistent Volume(s)    | `""`  |


### Common parameters

| Name                     | Description                                                                             | Value           |
| ------------------------ | --------------------------------------------------------------------------------------- | --------------- |
| `kubeVersion`            | Override Kubernetes version                                                             | `""`            |
| `nameOverride`           | String to partially override common.names.fullname                                      | `""`            |
| `fullnameOverride`       | String to fully override common.names.fullname                                          | `""`            |
| `commonLabels`           | Labels to add to all deployed objects                                                   | `{}`            |
| `commonAnnotations`      | Annotations to add to all deployed objects                                              | `{}`            |
| `clusterDomain`          | Kubernetes cluster domain name                                                          | `cluster.local` |
| `extraDeploy`            | Array of extra objects to deploy with the release                                       | `[]`            |
| `diagnosticMode.enabled` | Enable diagnostic mode (all probes will be disabled and the command will be overridden) | `false`         |
| `diagnosticMode.command` | Command to override all containers in the deployment                                    | `["sleep"]`     |
| `diagnosticMode.args`    | Args to override all containers in the deployment                                       | `["infinity"]`  |


### controller Parameters

| Name                                               | Description                                                                                          | Value                |
| -------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | -------------------- |
| `controller.image.registry`                        | controller image registry                                                                            | `docker.io`          |
| `controller.image.repository`                      | controller image repository                                                                          | `kubegems/kubegems`  |
| `controller.image.tag`                             | controller image tag (immutable tags are recommended)                                                | `latest`             |
| `controller.image.pullPolicy`                      | controller image pull policy                                                                         | `Always`             |
| `controller.image.pullSecrets`                     | controller image pull secrets                                                                        | `[]`                 |
| `controller.image.debug`                           | Enable controller image debug mode                                                                   | `false`              |
| `controller.replicaCount`                          | Number of controller replicas to deploy                                                              | `1`                  |
| `controller.containerPorts.probe`                  | controller probe container port                                                                      | `8080`               |
| `controller.livenessProbe.enabled`                 | Enable livenessProbe on controller containers                                                        | `true`               |
| `controller.livenessProbe.initialDelaySeconds`     | Initial delay seconds for livenessProbe                                                              | `5`                  |
| `controller.livenessProbe.periodSeconds`           | Period seconds for livenessProbe                                                                     | `10`                 |
| `controller.livenessProbe.timeoutSeconds`          | Timeout seconds for livenessProbe                                                                    | `5`                  |
| `controller.livenessProbe.failureThreshold`        | Failure threshold for livenessProbe                                                                  | `6`                  |
| `controller.livenessProbe.successThreshold`        | Success threshold for livenessProbe                                                                  | `1`                  |
| `controller.readinessProbe.enabled`                | Enable readinessProbe on controller containers                                                       | `true`               |
| `controller.readinessProbe.initialDelaySeconds`    | Initial delay seconds for readinessProbe                                                             | `5`                  |
| `controller.readinessProbe.periodSeconds`          | Period seconds for readinessProbe                                                                    | `10`                 |
| `controller.readinessProbe.timeoutSeconds`         | Timeout seconds for readinessProbe                                                                   | `5`                  |
| `controller.readinessProbe.failureThreshold`       | Failure threshold for readinessProbe                                                                 | `6`                  |
| `controller.readinessProbe.successThreshold`       | Success threshold for readinessProbe                                                                 | `1`                  |
| `controller.startupProbe.enabled`                  | Enable startupProbe on controller containers                                                         | `false`              |
| `controller.startupProbe.initialDelaySeconds`      | Initial delay seconds for startupProbe                                                               | `5`                  |
| `controller.startupProbe.periodSeconds`            | Period seconds for startupProbe                                                                      | `10`                 |
| `controller.startupProbe.timeoutSeconds`           | Timeout seconds for startupProbe                                                                     | `5`                  |
| `controller.startupProbe.failureThreshold`         | Failure threshold for startupProbe                                                                   | `6`                  |
| `controller.startupProbe.successThreshold`         | Success threshold for startupProbe                                                                   | `1`                  |
| `controller.customLivenessProbe`                   | Custom livenessProbe that overrides the default one                                                  | `{}`                 |
| `controller.customReadinessProbe`                  | Custom readinessProbe that overrides the default one                                                 | `{}`                 |
| `controller.customStartupProbe`                    | Custom startupProbe that overrides the default one                                                   | `{}`                 |
| `controller.resources.limits`                      | The resources limits for the controller containers                                                   | `{}`                 |
| `controller.resources.requests`                    | The requested resources for the controller containers                                                | `{}`                 |
| `controller.podSecurityContext.enabled`            | Enabled controller pods' Security Context                                                            | `false`              |
| `controller.podSecurityContext.fsGroup`            | Set controller pod's Security Context fsGroup                                                        | `1001`               |
| `controller.containerSecurityContext.enabled`      | Enabled controller containers' Security Context                                                      | `false`              |
| `controller.containerSecurityContext.runAsUser`    | Set controller containers' Security Context runAsUser                                                | `1001`               |
| `controller.containerSecurityContext.runAsNonRoot` | Set controller containers' Security Context runAsNonRoot                                             | `true`               |
| `controller.leaderElection.enabled`                | Enable leader election                                                                               | `true`               |
| `controller.baseDomain`                            | Models Ingress Base Domain                                                                           | `models.kubegems.io` |
| `controller.logLevel`                              | Log level                                                                                            | `debug`              |
| `controller.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for controller                      | `nil`                |
| `controller.command`                               | Override default container command (useful when using custom images)                                 | `[]`                 |
| `controller.args`                                  | Override default container args (useful when using custom images)                                    | `[]`                 |
| `controller.hostAliases`                           | controller pods host aliases                                                                         | `[]`                 |
| `controller.podLabels`                             | Extra labels for controller pods                                                                     | `{}`                 |
| `controller.podAnnotations`                        | Annotations for controller pods                                                                      | `{}`                 |
| `controller.podAffinityPreset`                     | Pod affinity preset. Ignored if `controller.affinity` is set. Allowed values: `soft` or `hard`       | `""`                 |
| `controller.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `controller.affinity` is set. Allowed values: `soft` or `hard`  | `soft`               |
| `controller.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `controller.affinity` is set. Allowed values: `soft` or `hard` | `""`                 |
| `controller.nodeAffinityPreset.key`                | Node label key to match. Ignored if `controller.affinity` is set                                     | `""`                 |
| `controller.nodeAffinityPreset.values`             | Node label values to match. Ignored if `controller.affinity` is set                                  | `[]`                 |
| `controller.enableAffinity`                        | If enabled Affinity for controller pods assignment                                                   | `false`              |
| `controller.affinity`                              | Affinity for controller pods assignment                                                              | `{}`                 |
| `controller.nodeSelector`                          | Node labels for controller pods assignment                                                           | `{}`                 |
| `controller.tolerations`                           | Tolerations for controller pods assignment                                                           | `[]`                 |
| `controller.updateStrategy.type`                   | controller statefulset strategy type                                                                 | `RollingUpdate`      |
| `controller.priorityClassName`                     | controller pods' priorityClassName                                                                   | `""`                 |
| `controller.schedulerName`                         | Name of the k8s scheduler (other than default) for controller pods                                   | `""`                 |
| `controller.lifecycleHooks`                        | for the controller container(s) to automate configuration before or after startup                    | `{}`                 |
| `controller.extraEnvVars`                          | Array with extra environment variables to add to controller nodes                                    | `[]`                 |
| `controller.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for controller nodes                            | `nil`                |
| `controller.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for controller nodes                               | `nil`                |
| `controller.extraVolumes`                          | Optionally specify extra list of additional volumes for the controller pod(s)                        | `[]`                 |
| `controller.extraVolumeMounts`                     | Optionally specify extra list of additional volumeMounts for the controller container(s)             | `[]`                 |
| `controller.sidecars`                              | Add additional sidecar containers to the controller pod(s)                                           | `{}`                 |
| `controller.initContainers`                        | Add additional init containers to the controller pod(s)                                              | `{}`                 |


### Agent Metrics parameters

| Name                                                  | Description                                                                                     | Value                    |
| ----------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ------------------------ |
| `controller.metrics.enabled`                          | Create a service for accessing the metrics endpoint                                             | `true`                   |
| `controller.metrics.service.type`                     | controller metrics service type                                                                 | `ClusterIP`              |
| `controller.metrics.service.port`                     | controller metrics service HTTP port                                                            | `9100`                   |
| `controller.metrics.service.nodePort`                 | Node port for HTTP                                                                              | `""`                     |
| `controller.metrics.service.clusterIP`                | controller metrics service Cluster IP                                                           | `""`                     |
| `controller.metrics.service.extraPorts`               | Extra ports to expose (normally used with the `sidecar` value)                                  | `[]`                     |
| `controller.metrics.service.loadBalancerIP`           | controller metrics service Load Balancer IP                                                     | `""`                     |
| `controller.metrics.service.loadBalancerSourceRanges` | controller metrics service Load Balancer sources                                                | `[]`                     |
| `controller.metrics.service.externalTrafficPolicy`    | controller metrics service external traffic policy                                              | `Cluster`                |
| `controller.metrics.service.annotations`              | Additional custom annotations for controller metrics service                                    | `{}`                     |
| `controller.metrics.serviceMonitor.enabled`           | Specify if a servicemonitor will be deployed for prometheus-operator                            | `true`                   |
| `controller.metrics.serviceMonitor.jobLabel`          | Specify the jobLabel to use for the prometheus-operator                                         | `app.kubernetes.io/name` |
| `controller.metrics.serviceMonitor.honorLabels`       | Honor metrics labels                                                                            | `false`                  |
| `controller.metrics.serviceMonitor.selector`          | Prometheus instance selector labels                                                             | `{}`                     |
| `controller.metrics.serviceMonitor.scrapeTimeout`     | Timeout after which the scrape is ended                                                         | `""`                     |
| `controller.metrics.serviceMonitor.interval`          | Scrape interval. If not set, the Prometheus default scrape interval is used                     | `""`                     |
| `controller.metrics.serviceMonitor.additionalLabels`  | Used to pass Labels that are required by the installed Prometheus Operator                      | `{}`                     |
| `controller.metrics.serviceMonitor.metricRelabelings` | Specify additional relabeling of metrics                                                        | `{}`                     |
| `controller.metrics.serviceMonitor.relabelings`       | Specify general relabeling                                                                      | `{}`                     |
| `store.image.registry`                                | store image registry                                                                            | `docker.io`              |
| `store.image.repository`                              | store image repository                                                                          | `kubegems/kubegems`      |
| `store.image.tag`                                     | store image tag (immutable tags are recommended)                                                | `latest`                 |
| `store.image.pullPolicy`                              | store image pull policy                                                                         | `Always`                 |
| `store.image.pullSecrets`                             | store image pull secrets                                                                        | `[]`                     |
| `store.image.debug`                                   | Enable store image debug mode                                                                   | `false`                  |
| `store.replicaCount`                                  | Number of store replicas to deploy                                                              | `1`                      |
| `store.containerPorts.probe`                          | store probe container port                                                                      | `8080`                   |
| `store.livenessProbe.enabled`                         | Enable livenessProbe on store containers                                                        | `true`                   |
| `store.livenessProbe.initialDelaySeconds`             | Initial delay seconds for livenessProbe                                                         | `5`                      |
| `store.livenessProbe.periodSeconds`                   | Period seconds for livenessProbe                                                                | `10`                     |
| `store.livenessProbe.timeoutSeconds`                  | Timeout seconds for livenessProbe                                                               | `5`                      |
| `store.livenessProbe.failureThreshold`                | Failure threshold for livenessProbe                                                             | `6`                      |
| `store.livenessProbe.successThreshold`                | Success threshold for livenessProbe                                                             | `1`                      |
| `store.readinessProbe.enabled`                        | Enable readinessProbe on store containers                                                       | `true`                   |
| `store.readinessProbe.initialDelaySeconds`            | Initial delay seconds for readinessProbe                                                        | `5`                      |
| `store.readinessProbe.periodSeconds`                  | Period seconds for readinessProbe                                                               | `10`                     |
| `store.readinessProbe.timeoutSeconds`                 | Timeout seconds for readinessProbe                                                              | `5`                      |
| `store.readinessProbe.failureThreshold`               | Failure threshold for readinessProbe                                                            | `6`                      |
| `store.readinessProbe.successThreshold`               | Success threshold for readinessProbe                                                            | `1`                      |
| `store.startupProbe.enabled`                          | Enable startupProbe on store containers                                                         | `false`                  |
| `store.startupProbe.initialDelaySeconds`              | Initial delay seconds for startupProbe                                                          | `5`                      |
| `store.startupProbe.periodSeconds`                    | Period seconds for startupProbe                                                                 | `10`                     |
| `store.startupProbe.timeoutSeconds`                   | Timeout seconds for startupProbe                                                                | `5`                      |
| `store.startupProbe.failureThreshold`                 | Failure threshold for startupProbe                                                              | `6`                      |
| `store.startupProbe.successThreshold`                 | Success threshold for startupProbe                                                              | `1`                      |
| `store.customLivenessProbe`                           | Custom livenessProbe that overrides the default one                                             | `{}`                     |
| `store.customReadinessProbe`                          | Custom readinessProbe that overrides the default one                                            | `{}`                     |
| `store.customStartupProbe`                            | Custom startupProbe that overrides the default one                                              | `{}`                     |
| `store.resources.limits`                              | The resources limits for the store containers                                                   | `{}`                     |
| `store.resources.requests`                            | The requested resources for the store containers                                                | `{}`                     |
| `store.podSecurityContext.enabled`                    | Enabled store pods' Security Context                                                            | `false`                  |
| `store.podSecurityContext.fsGroup`                    | Set store pod's Security Context fsGroup                                                        | `1001`                   |
| `store.containerSecurityContext.enabled`              | Enabled store containers' Security Context                                                      | `false`                  |
| `store.containerSecurityContext.runAsUser`            | Set store containers' Security Context runAsUser                                                | `1001`                   |
| `store.containerSecurityContext.runAsNonRoot`         | Set store containers' Security Context runAsNonRoot                                             | `true`                   |
| `store.logLevel`                                      | Log level                                                                                       | `debug`                  |
| `store.existingConfigmap`                             | The name of an existing ConfigMap with your custom configuration for store                      | `nil`                    |
| `store.command`                                       | Override default container command (useful when using custom images)                            | `[]`                     |
| `store.args`                                          | Override default container args (useful when using custom images)                               | `[]`                     |
| `store.hostAliases`                                   | store pods host aliases                                                                         | `[]`                     |
| `store.podLabels`                                     | Extra labels for store pods                                                                     | `{}`                     |
| `store.podAnnotations`                                | Annotations for store pods                                                                      | `{}`                     |
| `store.podAffinityPreset`                             | Pod affinity preset. Ignored if `store.affinity` is set. Allowed values: `soft` or `hard`       | `""`                     |
| `store.podAntiAffinityPreset`                         | Pod anti-affinity preset. Ignored if `store.affinity` is set. Allowed values: `soft` or `hard`  | `soft`                   |
| `store.nodeAffinityPreset.type`                       | Node affinity preset type. Ignored if `store.affinity` is set. Allowed values: `soft` or `hard` | `""`                     |
| `store.nodeAffinityPreset.key`                        | Node label key to match. Ignored if `store.affinity` is set                                     | `""`                     |
| `store.nodeAffinityPreset.values`                     | Node label values to match. Ignored if `store.affinity` is set                                  | `[]`                     |
| `store.enableAffinity`                                | If enabled Affinity for store pods assignment                                                   | `false`                  |
| `store.affinity`                                      | Affinity for store pods assignment                                                              | `{}`                     |
| `store.nodeSelector`                                  | Node labels for store pods assignment                                                           | `{}`                     |
| `store.tolerations`                                   | Tolerations for store pods assignment                                                           | `[]`                     |
| `store.updateStrategy.type`                           | store statefulset strategy type                                                                 | `RollingUpdate`          |
| `store.priorityClassName`                             | store pods' priorityClassName                                                                   | `""`                     |
| `store.schedulerName`                                 | Name of the k8s scheduler (other than default) for store pods                                   | `""`                     |
| `store.lifecycleHooks`                                | for the store container(s) to automate configuration before or after startup                    | `{}`                     |
| `store.extraEnvVars`                                  | Array with extra environment variables to add to store nodes                                    | `[]`                     |
| `store.extraEnvVarsCM`                                | Name of existing ConfigMap containing extra env vars for store nodes                            | `nil`                    |
| `store.extraEnvVarsSecret`                            | Name of existing Secret containing extra env vars for store nodes                               | `nil`                    |
| `store.extraVolumes`                                  | Optionally specify extra list of additional volumes for the store pod(s)                        | `[]`                     |
| `store.extraVolumeMounts`                             | Optionally specify extra list of additional volumeMounts for the store container(s)             | `[]`                     |
| `store.sidecars`                                      | Add additional sidecar containers to the store pod(s)                                           | `{}`                     |
| `store.initContainers`                                | Add additional init containers to the store pod(s)                                              | `{}`                     |
| `store.service.type`                                  | store service type                                                                              | `ClusterIP`              |
| `store.service.ports.http`                            | store service HTTP port                                                                         | `8080`                   |
| `store.service.nodePorts.http`                        | Node port for HTTP                                                                              | `nil`                    |
| `store.service.clusterIP`                             | store service Cluster IP                                                                        | `nil`                    |
| `store.service.loadBalancerIP`                        | store service Load Balancer IP                                                                  | `nil`                    |
| `store.service.loadBalancerSourceRanges`              | store service Load Balancer sources                                                             | `[]`                     |
| `store.service.externalTrafficPolicy`                 | store service external traffic policy                                                           | `Cluster`                |
| `store.service.annotations`                           | Additional custom annotations for store service                                                 | `{}`                     |
| `store.service.extraPorts`                            | Extra ports to expose in store service (normally used with the `sidecars` value)                | `[]`                     |
| `sync.image.registry`                                 | sync image registry                                                                             | `docker.io`              |
| `sync.image.repository`                               | sync image repository                                                                           | `kubegems/ai-model-sync` |
| `sync.image.tag`                                      | sync image tag (immutable tags are recommended)                                                 | `v1.16`                  |
| `sync.image.pullPolicy`                               | sync image pull policy                                                                          | `IfNotPresent`           |
| `sync.image.pullSecrets`                              | sync image pull secrets                                                                         | `[]`                     |
| `sync.image.debug`                                    | Enable sync image debug mode                                                                    | `false`                  |
| `sync.replicaCount`                                   | Number of sync replicas to deploy                                                               | `1`                      |
| `sync.containerPorts.http`                            | http container port                                                                             | `8000`                   |
| `sync.livenessProbe.enabled`                          | Enable livenessProbe on sync containers                                                         | `true`                   |
| `sync.livenessProbe.initialDelaySeconds`              | Initial delay seconds for livenessProbe                                                         | `5`                      |
| `sync.livenessProbe.periodSeconds`                    | Period seconds for livenessProbe                                                                | `10`                     |
| `sync.livenessProbe.timeoutSeconds`                   | Timeout seconds for livenessProbe                                                               | `5`                      |
| `sync.livenessProbe.failureThreshold`                 | Failure threshold for livenessProbe                                                             | `6`                      |
| `sync.livenessProbe.successThreshold`                 | Success threshold for livenessProbe                                                             | `1`                      |
| `sync.readinessProbe.enabled`                         | Enable readinessProbe on sync containers                                                        | `true`                   |
| `sync.readinessProbe.initialDelaySeconds`             | Initial delay seconds for readinessProbe                                                        | `5`                      |
| `sync.readinessProbe.periodSeconds`                   | Period seconds for readinessProbe                                                               | `10`                     |
| `sync.readinessProbe.timeoutSeconds`                  | Timeout seconds for readinessProbe                                                              | `5`                      |
| `sync.readinessProbe.failureThreshold`                | Failure threshold for readinessProbe                                                            | `6`                      |
| `sync.readinessProbe.successThreshold`                | Success threshold for readinessProbe                                                            | `1`                      |
| `sync.startupProbe.enabled`                           | Enable startupProbe on sync containers                                                          | `false`                  |
| `sync.startupProbe.initialDelaySeconds`               | Initial delay seconds for startupProbe                                                          | `5`                      |
| `sync.startupProbe.periodSeconds`                     | Period seconds for startupProbe                                                                 | `10`                     |
| `sync.startupProbe.timeoutSeconds`                    | Timeout seconds for startupProbe                                                                | `5`                      |
| `sync.startupProbe.failureThreshold`                  | Failure threshold for startupProbe                                                              | `6`                      |
| `sync.startupProbe.successThreshold`                  | Success threshold for startupProbe                                                              | `1`                      |
| `sync.customLivenessProbe`                            | Custom livenessProbe that overrides the default one                                             | `{}`                     |
| `sync.customReadinessProbe`                           | Custom readinessProbe that overrides the default one                                            | `{}`                     |
| `sync.customStartupProbe`                             | Custom startupProbe that overrides the default one                                              | `{}`                     |
| `sync.resources.limits`                               | The resources limits for the sync containers                                                    | `{}`                     |
| `sync.resources.requests`                             | The requested resources for the sync containers                                                 | `{}`                     |
| `sync.podSecurityContext.enabled`                     | Enabled sync pods' Security Context                                                             | `false`                  |
| `sync.podSecurityContext.fsGroup`                     | Set sync pod's Security Context fsGroup                                                         | `1001`                   |
| `sync.containerSecurityContext.enabled`               | Enabled sync containers' Security Context                                                       | `false`                  |
| `sync.containerSecurityContext.runAsUser`             | Set sync containers' Security Context runAsUser                                                 | `1001`                   |
| `sync.containerSecurityContext.runAsNonRoot`          | Set sync containers' Security Context runAsNonRoot                                              | `true`                   |
| `sync.logLevel`                                       | Log level                                                                                       | `debug`                  |
| `sync.existingConfigmap`                              | The name of an existing ConfigMap with your custom configuration for sync                       | `nil`                    |
| `sync.command`                                        | Override default container command (useful when using custom images)                            | `[]`                     |
| `sync.args`                                           | Override default container args (useful when using custom images)                               | `[]`                     |
| `sync.hostAliases`                                    | sync pods host aliases                                                                          | `[]`                     |
| `sync.podLabels`                                      | Extra labels for sync pods                                                                      | `{}`                     |
| `sync.podAnnotations`                                 | Annotations for sync pods                                                                       | `{}`                     |
| `sync.podAffinityPreset`                              | Pod affinity preset. Ignored if `sync.affinity` is set. Allowed values: `soft` or `hard`        | `""`                     |
| `sync.podAntiAffinityPreset`                          | Pod anti-affinity preset. Ignored if `sync.affinity` is set. Allowed values: `soft` or `hard`   | `soft`                   |
| `sync.nodeAffinityPreset.type`                        | Node affinity preset type. Ignored if `sync.affinity` is set. Allowed values: `soft` or `hard`  | `""`                     |
| `sync.nodeAffinityPreset.key`                         | Node label key to match. Ignored if `sync.affinity` is set                                      | `""`                     |
| `sync.nodeAffinityPreset.values`                      | Node label values to match. Ignored if `sync.affinity` is set                                   | `[]`                     |
| `sync.enableAffinity`                                 | If enabled Affinity for sync pods assignment                                                    | `false`                  |
| `sync.affinity`                                       | Affinity for sync pods assignment                                                               | `{}`                     |
| `sync.nodeSelector`                                   | Node labels for sync pods assignment                                                            | `{}`                     |
| `sync.tolerations`                                    | Tolerations for sync pods assignment                                                            | `[]`                     |
| `sync.updateStrategy.type`                            | sync statefulset strategy type                                                                  | `RollingUpdate`          |
| `sync.priorityClassName`                              | sync pods' priorityClassName                                                                    | `""`                     |
| `sync.schedulerName`                                  | Name of the k8s scheduler (other than default) for sync pods                                    | `""`                     |
| `sync.lifecycleHooks`                                 | for the sync container(s) to automate configuration before or after startup                     | `{}`                     |
| `sync.extraEnvVars`                                   | Array with extra environment variables to add to sync nodes                                     | `[]`                     |
| `sync.extraEnvVarsCM`                                 | Name of existing ConfigMap containing extra env vars for sync nodes                             | `nil`                    |
| `sync.extraEnvVarsSecret`                             | Name of existing Secret containing extra env vars for sync nodes                                | `nil`                    |
| `sync.extraVolumes`                                   | Optionally specify extra list of additional volumes for the sync pod(s)                         | `[]`                     |
| `sync.extraVolumeMounts`                              | Optionally specify extra list of additional volumeMounts for the sync container(s)              | `[]`                     |
| `sync.sidecars`                                       | Add additional sidecar containers to the sync pod(s)                                            | `{}`                     |
| `sync.initContainers`                                 | Add additional init containers to the sync pod(s)                                               | `{}`                     |
| `sync.service.type`                                   | sync service type                                                                               | `ClusterIP`              |
| `sync.service.ports.http`                             | sync service HTTP port                                                                          | `8080`                   |
| `sync.service.nodePorts.http`                         | Node port for HTTP                                                                              | `nil`                    |
| `sync.service.clusterIP`                              | sync service Cluster IP                                                                         | `nil`                    |
| `sync.service.loadBalancerIP`                         | sync service Load Balancer IP                                                                   | `nil`                    |
| `sync.service.loadBalancerSourceRanges`               | sync service Load Balancer sources                                                              | `[]`                     |
| `sync.service.externalTrafficPolicy`                  | sync service external traffic policy                                                            | `Cluster`                |
| `sync.service.annotations`                            | Additional custom annotations for sync service                                                  | `{}`                     |
| `sync.service.extraPorts`                             | Extra ports to expose in sync service (normally used with the `sidecars` value)                 | `[]`                     |


### MongoDB parameters

| Name                       | Description               | Value              |
| -------------------------- | ------------------------- | ------------------ |
| `mongodb.auth`             | auth of mongo             | `{}`               |
| `mongodb.image.repository` | mongo db image repository | `kubegems/mongodb` |


### RBAC Parameters

| Name                    | Description                                          | Value  |
| ----------------------- | ---------------------------------------------------- | ------ |
| `rbac.create`           | Specifies whether RBAC resources should be created   | `true` |
| `serviceAccount.create` | Specifies whether a ServiceAccount should be created | `true` |
| `serviceAccount.name`   | The name of the ServiceAccount to use.               | `""`   |


Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
helm install my-release \
  --set kubegems-installerUsername=admin \
  --set kubegems-installerPassword=password \
  --set mariadb.auth.rootPassword=secretpassword \
    kubegems/kubegems-installer
```

The above command sets the kubegems-installer administrator account username and password to `admin` and `password` respectively. Additionally, it sets the MariaDB `root` user password to `secretpassword`.

> NOTE: Once this chart is deployed, it is not possible to change the application's access credentials, such as usernames or passwords, using Helm. To change these application credentials after deployment, delete any persistent volumes (PVs) used by the chart and re-deploy it, or use the application's built-in administrative tools if available.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
helm install my-release -f values.yaml kubegems/kubegems-installer
```

> **Tip**: You can use the default [values.yaml](values.yaml)

## Configuration and installation details

### External database support

%%IF NEEDED%%

You may want to have kubegems-installer connect to an external database rather than installing one inside your cluster. Typical reasons for this are to use a managed database service, or to share a common database server for all your applications. To achieve this, the chart allows you to specify credentials for an external database with the [`externalDatabase` parameter](#parameters). You should also disable the MariaDB installation with the `mariadb.enabled` option. Here is an example:

```console
mariadb.enabled=false
externalDatabase.host=myexternalhost
externalDatabase.user=myuser
externalDatabase.password=mypassword
externalDatabase.database=mydatabase
externalDatabase.port=3306
```
### Additional environment variables

In case you want to add extra environment variables (useful for advanced operations like custom init scripts), you can use the `extraEnvVars` property.

```yaml
kubegems-installer:
  extraEnvVars:
    - name: LOG_LEVEL
      value: error
```

Alternatively, you can use a ConfigMap or a Secret with the environment variables. To do so, use the `extraEnvVarsCM` or the `extraEnvVarsSecret` values.

## License

Copyright &copy; 2022 KubeGems.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
