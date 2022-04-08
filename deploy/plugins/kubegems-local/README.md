# kubegems-local

KubeGems local components.(check existing examples)

## TL;DR

```console
$ helm install my-release kubegems-local
```

## Introduction

kubegems local components.

## Prerequisites

- Kubernetes 1.18+
- Helm 3.2.0+

## Installing the Chart

To install the chart with the release name `my-release`:

```console
helm install my-release bitnami/kubegems-local
```

The command deploys kubegems-local on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

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
| `diagnosticMode.command` | Command to override all containers in the the deployment(s)/statefulset(s)              | `["sleep"]`     |
| `diagnosticMode.args`    | Args to override all containers in the the deployment(s)/statefulset(s)                 | `["infinity"]`  |


### Agent Parameters

| Name                                          | Description                                                                                     | Value               |
| --------------------------------------------- | ----------------------------------------------------------------------------------------------- | ------------------- |
| `agent.image.registry`                        | agent image registry                                                                            | `docker.io`         |
| `agent.image.repository`                      | agent image repository                                                                          | `kubegems/kubegems` |
| `agent.image.tag`                             | agent image tag (immutable tags are recommended)                                                | `latest`            |
| `agent.image.pullPolicy`                      | agent image pull policy                                                                         | `IfNotPresent`      |
| `agent.image.pullSecrets`                     | agent image pull secrets                                                                        | `[]`                |
| `agent.image.debug`                           | Enable agent image debug mode                                                                   | `false`             |
| `agent.replicaCount`                          | Number of agent replicas to deploy                                                              | `1`                 |
| `agent.containerPorts.http`                   | agent HTTP container port                                                                       | `8080`              |
| `agent.livenessProbe.enabled`                 | Enable livenessProbe on agent containers                                                        | `true`              |
| `agent.livenessProbe.initialDelaySeconds`     | Initial delay seconds for livenessProbe                                                         | `10`                |
| `agent.livenessProbe.periodSeconds`           | Period seconds for livenessProbe                                                                | `20`                |
| `agent.livenessProbe.timeoutSeconds`          | Timeout seconds for livenessProbe                                                               | `1`                 |
| `agent.livenessProbe.failureThreshold`        | Failure threshold for livenessProbe                                                             | `6`                 |
| `agent.livenessProbe.successThreshold`        | Success threshold for livenessProbe                                                             | `1`                 |
| `agent.readinessProbe.enabled`                | Enable readinessProbe on agent containers                                                       | `true`              |
| `agent.readinessProbe.initialDelaySeconds`    | Initial delay seconds for readinessProbe                                                        | `10`                |
| `agent.readinessProbe.periodSeconds`          | Period seconds for readinessProbe                                                               | `20`                |
| `agent.readinessProbe.timeoutSeconds`         | Timeout seconds for readinessProbe                                                              | `1`                 |
| `agent.readinessProbe.failureThreshold`       | Failure threshold for readinessProbe                                                            | `6`                 |
| `agent.readinessProbe.successThreshold`       | Success threshold for readinessProbe                                                            | `1`                 |
| `agent.startupProbe.enabled`                  | Enable startupProbe on agent containers                                                         | `false`             |
| `agent.startupProbe.initialDelaySeconds`      | Initial delay seconds for startupProbe                                                          | `10`                |
| `agent.startupProbe.periodSeconds`            | Period seconds for startupProbe                                                                 | `20`                |
| `agent.startupProbe.timeoutSeconds`           | Timeout seconds for startupProbe                                                                | `1`                 |
| `agent.startupProbe.failureThreshold`         | Failure threshold for startupProbe                                                              | `6`                 |
| `agent.startupProbe.successThreshold`         | Success threshold for startupProbe                                                              | `1`                 |
| `agent.customLivenessProbe`                   | Custom livenessProbe that overrides the default one                                             | `{}`                |
| `agent.customReadinessProbe`                  | Custom readinessProbe that overrides the default one                                            | `{}`                |
| `agent.customStartupProbe`                    | Custom startupProbe that overrides the default one                                              | `{}`                |
| `agent.resources.limits`                      | The resources limits for the agent containers                                                   | `{}`                |
| `agent.resources.requests`                    | The requested resources for the agent containers                                                | `{}`                |
| `agent.podSecurityContext.enabled`            | Enabled agent pods' Security Context                                                            | `true`              |
| `agent.podSecurityContext.fsGroup`            | Set agent pod's Security Context fsGroup                                                        | `1001`              |
| `agent.containerSecurityContext.enabled`      | Enabled agent containers' Security Context                                                      | `true`              |
| `agent.containerSecurityContext.runAsUser`    | Set agent containers' Security Context runAsUser                                                | `1001`              |
| `agent.containerSecurityContext.runAsNonRoot` | Set agent containers' Security Context runAsNonRoot                                             | `true`              |
| `agent.tls.enabled`                           | Enable agent http listen TLS                                                                    | `true`              |
| `agent.tls.useCertManager`                    | using cert manager to generate tls secret,if not using self-signed certificates if not exists   | `true`              |
| `agent.tls.secretName`                        | customize default tls secret name                                                               | `""`                |
| `agent.httpSignature.enabled`                 | Enable agent HTTP Signature validation                                                          | `true`              |
| `agent.httpSignature.token`                   | HTTP Signature encode token override                                                            | `""`                |
| `agent.logLevel`                              | Set log level                                                                                   | `info`              |
| `agent.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for agent                      | `nil`               |
| `agent.command`                               | Override default container command (useful when using custom images)                            | `[]`                |
| `agent.args`                                  | Override default container args (useful when using custom images)                               | `[]`                |
| `agent.hostAliases`                           | agent pods host aliases                                                                         | `[]`                |
| `agent.podLabels`                             | Extra labels for agent pods                                                                     | `{}`                |
| `agent.podAnnotations`                        | Annotations for agent pods                                                                      | `{}`                |
| `agent.podAffinityPreset`                     | Pod affinity preset. Ignored if `agent.affinity` is set. Allowed values: `soft` or `hard`       | `""`                |
| `agent.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `agent.affinity` is set. Allowed values: `soft` or `hard`  | `soft`              |
| `agent.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `agent.affinity` is set. Allowed values: `soft` or `hard` | `""`                |
| `agent.nodeAffinityPreset.key`                | Node label key to match. Ignored if `agent.affinity` is set                                     | `""`                |
| `agent.nodeAffinityPreset.values`             | Node label values to match. Ignored if `agent.affinity` is set                                  | `[]`                |
| `agent.affinity`                              | Affinity for agent pods assignment                                                              | `{}`                |
| `agent.nodeSelector`                          | Node labels for agent pods assignment                                                           | `{}`                |
| `agent.tolerations`                           | Tolerations for agent pods assignment                                                           | `[]`                |
| `agent.updateStrategy.type`                   | agent statefulset strategy type                                                                 | `RollingUpdate`     |
| `agent.priorityClassName`                     | agent pods' priorityClassName                                                                   | `""`                |
| `agent.schedulerName`                         | Name of the k8s scheduler (other than default) for agent pods                                   | `""`                |
| `agent.lifecycleHooks`                        | for the agent container(s) to automate configuration before or after startup                    | `{}`                |
| `agent.extraEnvVars`                          | Array with extra environment variables to add to agent nodes                                    | `[]`                |
| `agent.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for agent nodes                            | `nil`               |
| `agent.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for agent nodes                               | `nil`               |
| `agent.extraVolumes`                          | Optionally specify extra list of additional volumes for the agent pod(s)                        | `[]`                |
| `agent.extraVolumeMounts`                     | Optionally specify extra list of additional volumeMounts for the agent container(s)             | `[]`                |
| `agent.sidecars`                              | Add additional sidecar containers to the agent pod(s)                                           | `{}`                |
| `agent.initContainers`                        | Add additional init containers to the agent pod(s)                                              | `{}`                |


### Agent RBAC Parameters

| Name                                     | Description                                                                      | Value       |
| ---------------------------------------- | -------------------------------------------------------------------------------- | ----------- |
| `agent.rbac.create`                      | Specifies whether RBAC resources should be created                               | `true`      |
| `agent.rbac.singleNamespace`             | limit agent scope in a single namespace                                          | `false`     |
| `agent.serviceAccount.create`            | Specifies whether a ServiceAccount should be created                             | `true`      |
| `agent.serviceAccount.name`              | The name of the ServiceAccount to use.                                           | `""`        |
| `agent.service.type`                     | agent service type                                                               | `ClusterIP` |
| `agent.service.ports.http`               | agent service HTTP port                                                          | `8041`      |
| `agent.service.nodePorts.http`           | Node port for HTTP                                                               | `nil`       |
| `agent.service.clusterIP`                | agent service Cluster IP                                                         | `nil`       |
| `agent.service.loadBalancerIP`           | agent service Load Balancer IP                                                   | `nil`       |
| `agent.service.loadBalancerSourceRanges` | agent service Load Balancer sources                                              | `[]`        |
| `agent.service.externalTrafficPolicy`    | agent service external traffic policy                                            | `Cluster`   |
| `agent.service.annotations`              | Additional custom annotations for agent service                                  | `{}`        |
| `agent.service.extraPorts`               | Extra ports to expose in agent service (normally used with the `sidecars` value) | `[]`        |


### Agent Metrics parameters

| Name                                             | Description                                                                 | Value                    |
| ------------------------------------------------ | --------------------------------------------------------------------------- | ------------------------ |
| `agent.metrics.enabled`                          | Create a service for accessing the metrics endpoint                         | `true`                   |
| `agent.metrics.service.type`                     | controller metrics service type                                             | `ClusterIP`              |
| `agent.metrics.service.port`                     | controller metrics service HTTP port                                        | `9100`                   |
| `agent.metrics.service.nodePort`                 | Node port for HTTP                                                          | `""`                     |
| `agent.metrics.service.clusterIP`                | controller metrics service Cluster IP                                       | `""`                     |
| `agent.metrics.service.extraPorts`               | Extra ports to expose (normally used with the `sidecar` value)              | `[]`                     |
| `agent.metrics.service.loadBalancerIP`           | controller metrics service Load Balancer IP                                 | `""`                     |
| `agent.metrics.service.loadBalancerSourceRanges` | controller metrics service Load Balancer sources                            | `[]`                     |
| `agent.metrics.service.externalTrafficPolicy`    | controller metrics service external traffic policy                          | `Cluster`                |
| `agent.metrics.service.annotations`              | Additional custom annotations for controller metrics service                | `{}`                     |
| `agent.metrics.serviceMonitor.enabled`           | Specify if a servicemonitor will be deployed for prometheus-operator        | `true`                   |
| `agent.metrics.serviceMonitor.jobLabel`          | Specify the jobLabel to use for the prometheus-operator                     | `app.kubernetes.io/name` |
| `agent.metrics.serviceMonitor.honorLabels`       | Honor metrics labels                                                        | `false`                  |
| `agent.metrics.serviceMonitor.selector`          | Prometheus instance selector labels                                         | `{}`                     |
| `agent.metrics.serviceMonitor.scrapeTimeout`     | Timeout after which the scrape is ended                                     | `""`                     |
| `agent.metrics.serviceMonitor.interval`          | Scrape interval. If not set, the Prometheus default scrape interval is used | `""`                     |
| `agent.metrics.serviceMonitor.additionalLabels`  | Used to pass Labels that are required by the installed Prometheus Operator  | `{}`                     |
| `agent.metrics.serviceMonitor.metricRelabelings` | Specify additional relabeling of metrics                                    | `undefined`              |
| `agent.metrics.serviceMonitor.relabelings`       | Specify general relabeling                                                  | `undefined`              |


### Controller Parameters

| Name                                               | Description                                                                                          | Value               |
| -------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ------------------- |
| `controller.image.registry`                        | controller image registry                                                                            | `docker.io`         |
| `controller.image.repository`                      | controller image repository                                                                          | `kubegems/kubegems` |
| `controller.image.tag`                             | controller image tag (immutable tags are recommended)                                                | `latest`            |
| `controller.image.pullPolicy`                      | controller image pull policy                                                                         | `IfNotPresent`      |
| `controller.image.pullSecrets`                     | controller image pull secrets                                                                        | `[]`                |
| `controller.image.debug`                           | Enable controller image debug mode                                                                   | `false`             |
| `controller.replicaCount`                          | Number of controller replicas to deploy                                                              | `1`                 |
| `controller.containerPorts.webhook`                | controller webhook port                                                                              | `443`               |
| `controller.containerPorts.probe`                  | controller probe port                                                                                | `8080`              |
| `controller.livenessProbe.enabled`                 | Enable livenessProbe on controller containers                                                        | `true`              |
| `controller.livenessProbe.initialDelaySeconds`     | Initial delay seconds for livenessProbe                                                              | `10`                |
| `controller.livenessProbe.periodSeconds`           | Period seconds for livenessProbe                                                                     | `20`                |
| `controller.livenessProbe.timeoutSeconds`          | Timeout seconds for livenessProbe                                                                    | `1`                 |
| `controller.livenessProbe.failureThreshold`        | Failure threshold for livenessProbe                                                                  | `6`                 |
| `controller.livenessProbe.successThreshold`        | Success threshold for livenessProbe                                                                  | `1`                 |
| `controller.readinessProbe.enabled`                | Enable readinessProbe on controller containers                                                       | `true`              |
| `controller.readinessProbe.initialDelaySeconds`    | Initial delay seconds for readinessProbe                                                             | `10`                |
| `controller.readinessProbe.periodSeconds`          | Period seconds for readinessProbe                                                                    | `20`                |
| `controller.readinessProbe.timeoutSeconds`         | Timeout seconds for readinessProbe                                                                   | `1`                 |
| `controller.readinessProbe.failureThreshold`       | Failure threshold for readinessProbe                                                                 | `6`                 |
| `controller.readinessProbe.successThreshold`       | Success threshold for readinessProbe                                                                 | `1`                 |
| `controller.startupProbe.enabled`                  | Enable startupProbe on controller containers                                                         | `false`             |
| `controller.startupProbe.initialDelaySeconds`      | Initial delay seconds for startupProbe                                                               | `10`                |
| `controller.startupProbe.periodSeconds`            | Period seconds for startupProbe                                                                      | `20`                |
| `controller.startupProbe.timeoutSeconds`           | Timeout seconds for startupProbe                                                                     | `1`                 |
| `controller.startupProbe.failureThreshold`         | Failure threshold for startupProbe                                                                   | `6`                 |
| `controller.startupProbe.successThreshold`         | Success threshold for startupProbe                                                                   | `1`                 |
| `controller.customLivenessProbe`                   | Custom livenessProbe that overrides the default one                                                  | `{}`                |
| `controller.customReadinessProbe`                  | Custom readinessProbe that overrides the default one                                                 | `{}`                |
| `controller.customStartupProbe`                    | Custom startupProbe that overrides the default one                                                   | `{}`                |
| `controller.resources.limits`                      | The resources limits for the controller containers                                                   | `{}`                |
| `controller.resources.requests`                    | The requested resources for the controller containers                                                | `{}`                |
| `controller.podSecurityContext.enabled`            | Enabled controller pods' Security Context                                                            | `true`              |
| `controller.podSecurityContext.fsGroup`            | Set controller pod's Security Context fsGroup                                                        | `1001`              |
| `controller.containerSecurityContext.enabled`      | Enabled controller containers' Security Context                                                      | `true`              |
| `controller.containerSecurityContext.runAsUser`    | Set controller containers' Security Context runAsUser                                                | `1001`              |
| `controller.containerSecurityContext.runAsNonRoot` | Set controller containers' Security Context runAsNonRoot                                             | `true`              |
| `controller.leaderElection.enabled`                | Enable leader election for controller                                                                | `true`              |
| `controller.logLevel`                              | Log level for controller                                                                             | `info`              |
| `controller.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for controller                      | `nil`               |
| `controller.command`                               | Override default container command (useful when using custom images)                                 | `[]`                |
| `controller.args`                                  | Override default container args (useful when using custom images)                                    | `[]`                |
| `controller.hostAliases`                           | controller pods host aliases                                                                         | `[]`                |
| `controller.podLabels`                             | Extra labels for controller pods                                                                     | `{}`                |
| `controller.podAnnotations`                        | Annotations for controller pods                                                                      | `{}`                |
| `controller.podAffinityPreset`                     | Pod affinity preset. Ignored if `controller.affinity` is set. Allowed values: `soft` or `hard`       | `""`                |
| `controller.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `controller.affinity` is set. Allowed values: `soft` or `hard`  | `soft`              |
| `controller.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `controller.affinity` is set. Allowed values: `soft` or `hard` | `""`                |
| `controller.nodeAffinityPreset.key`                | Node label key to match. Ignored if `controller.affinity` is set                                     | `""`                |
| `controller.nodeAffinityPreset.values`             | Node label values to match. Ignored if `controller.affinity` is set                                  | `[]`                |
| `controller.affinity`                              | Affinity for controller pods assignment                                                              | `{}`                |
| `controller.nodeSelector`                          | Node labels for controller pods assignment                                                           | `{}`                |
| `controller.tolerations`                           | Tolerations for controller pods assignment                                                           | `[]`                |
| `controller.updateStrategy.type`                   | controller statefulset strategy type                                                                 | `RollingUpdate`     |
| `controller.priorityClassName`                     | controller pods' priorityClassName                                                                   | `""`                |
| `controller.schedulerName`                         | Name of the k8s scheduler (other than default) for controller pods                                   | `""`                |
| `controller.lifecycleHooks`                        | for the controller container(s) to automate configuration before or after startup                    | `{}`                |
| `controller.extraEnvVars`                          | Array with extra environment variables to add to controller nodes                                    | `[]`                |
| `controller.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for controller nodes                            | `nil`               |
| `controller.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for controller nodes                               | `nil`               |
| `controller.extraVolumes`                          | Optionally specify extra list of additional volumes for the controller pod(s)                        | `[]`                |
| `controller.extraVolumeMounts`                     | Optionally specify extra list of additional volumeMounts for the controller container(s)             | `[]`                |
| `controller.sidecars`                              | Add additional sidecar containers to the controller pod(s)                                           | `{}`                |
| `controller.initContainers`                        | Add additional init containers to the controller pod(s)                                              | `{}`                |


### Controller RBAC Parameters

| Name                               | Description                                          | Value  |
| ---------------------------------- | ---------------------------------------------------- | ------ |
| `controller.rbac.create`           | Specifies whether RBAC resources should be created   | `true` |
| `controller.serviceAccount.create` | Specifies whether a ServiceAccount should be created | `true` |
| `controller.serviceAccount.name`   | The name of the ServiceAccount to use.               | `""`   |


### Controller Webhook Parameters

| Name                                                  | Description                                                                      | Value       |
| ----------------------------------------------------- | -------------------------------------------------------------------------------- | ----------- |
| `controller.webhook.enabled`                          | Specifies whether the webhook should be enabled                                  | `true`      |
| `controller.webhook.useCertManager`                   | using cert-manager to generate a  certificate                                    | `true`      |
| `controller.webhook.secretName`                       | tls secret name for webhook                                                      | `""`        |
| `controller.webhook.service.type`                     | agent service type                                                               | `ClusterIP` |
| `controller.webhook.service.ports.http`               | webhook service HTTP port                                                        | `443`       |
| `controller.webhook.service.nodePorts.http`           | Node port for HTTP                                                               | `nil`       |
| `controller.webhook.service.clusterIP`                | agent service Cluster IP                                                         | `nil`       |
| `controller.webhook.service.loadBalancerIP`           | agent service Load Balancer IP                                                   | `nil`       |
| `controller.webhook.service.loadBalancerSourceRanges` | agent service Load Balancer sources                                              | `[]`        |
| `controller.webhook.service.externalTrafficPolicy`    | agent service external traffic policy                                            | `Cluster`   |
| `controller.webhook.service.annotations`              | Additional custom annotations for agent service                                  | `{}`        |
| `controller.webhook.service.extraPorts`               | Extra ports to expose in agent service (normally used with the `sidecars` value) | `[]`        |


### Controller Metrics parameters

| Name                                                  | Description                                                                 | Value                    |
| ----------------------------------------------------- | --------------------------------------------------------------------------- | ------------------------ |
| `controller.metrics.enabled`                          | Create a service for accessing the metrics endpoint                         | `true`                   |
| `controller.metrics.service.type`                     | controller metrics service type                                             | `ClusterIP`              |
| `controller.metrics.service.port`                     | controller metrics service HTTP port                                        | `9100`                   |
| `controller.metrics.service.nodePort`                 | Node port for HTTP                                                          | `""`                     |
| `controller.metrics.service.clusterIP`                | controller metrics service Cluster IP                                       | `""`                     |
| `controller.metrics.service.extraPorts`               | Extra ports to expose (normally used with the `sidecar` value)              | `[]`                     |
| `controller.metrics.service.loadBalancerIP`           | controller metrics service Load Balancer IP                                 | `""`                     |
| `controller.metrics.service.loadBalancerSourceRanges` | controller metrics service Load Balancer sources                            | `[]`                     |
| `controller.metrics.service.externalTrafficPolicy`    | controller metrics service external traffic policy                          | `Cluster`                |
| `controller.metrics.service.annotations`              | Additional custom annotations for controller metrics service                | `{}`                     |
| `controller.metrics.serviceMonitor.enabled`           | Specify if a servicemonitor will be deployed for prometheus-operator        | `true`                   |
| `controller.metrics.serviceMonitor.jobLabel`          | Specify the jobLabel to use for the prometheus-operator                     | `app.kubernetes.io/name` |
| `controller.metrics.serviceMonitor.honorLabels`       | Honor metrics labels                                                        | `false`                  |
| `controller.metrics.serviceMonitor.selector`          | Prometheus instance selector labels                                         | `{}`                     |
| `controller.metrics.serviceMonitor.scrapeTimeout`     | Timeout after which the scrape is ended                                     | `""`                     |
| `controller.metrics.serviceMonitor.interval`          | Scrape interval. If not set, the Prometheus default scrape interval is used | `""`                     |
| `controller.metrics.serviceMonitor.additionalLabels`  | Used to pass Labels that are required by the installed Prometheus Operator  | `{}`                     |
| `controller.metrics.serviceMonitor.metricRelabelings` | Specify additional relabeling of metrics                                    | `[]`                     |
| `controller.metrics.serviceMonitor.relabelings`       | Specify general relabeling                                                  | `[]`                     |


### Traffic Exposure Parameters

| Name                  | Description                                                                                                                      | Value                    |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------------------ |
| `ingress.enabled`     | Enable ingress record generation for agent                                                                                       | `false`                  |
| `ingress.pathType`    | Ingress path type                                                                                                                | `ImplementationSpecific` |
| `ingress.apiVersion`  | Force Ingress API version (automatically detected if not set)                                                                    | `nil`                    |
| `ingress.hostname`    | Default host for the ingress record                                                                                              | `kubegems-local.local`   |
| `ingress.path`        | Default path for the ingress record                                                                                              | `/`                      |
| `ingress.annotations` | Additional annotations for the Ingress resource. To enable certificate autogeneration, place here your cert-manager annotations. | `{}`                     |
| `ingress.tls`         | Enable TLS configuration for the host defined at `ingress.hostname` parameter                                                    | `false`                  |
| `ingress.selfSigned`  | Create a TLS secret for this ingress record using self-signed certificates generated by Helm                                     | `false`                  |
| `ingress.extraHosts`  | An array with additional hostname(s) to be covered with the ingress record                                                       | `[]`                     |
| `ingress.extraPaths`  | An array with additional arbitrary paths that may need to be added to the ingress under the main host                            | `[]`                     |
| `ingress.extraTls`    | TLS configuration for additional hostname(s) to be covered with this ingress record                                              | `[]`                     |
| `ingress.secrets`     | Custom TLS certificates as secrets                                                                                               | `[]`                     |


See https://github.com/bitnami-labs/readme-generator-for-helm to create the table

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
helm install my-release kubegems-local
```

> NOTE: Once this chart is deployed, it is not possible to change the application's access credentials, such as usernames or passwords, using Helm. To change these application credentials after deployment, delete any persistent volumes (PVs) used by the chart and re-deploy it, or use the application's built-in administrative tools if available.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
helm install my-release -f values.yaml kubegems-local
```

> **Tip**: You can use the default [values.yaml](values.yaml)

## Configuration and installation details

### [Rolling VS Immutable tags](https://docs.bitnami.com/containers/how-to/understand-rolling-tags-containers/)

It is strongly recommended to use immutable tags in a production environment. This ensures your deployment does not change automatically if the same tag is updated with a different image.

Bitnami will release a new chart updating its containers if a new version of the main container, significant changes, or critical vulnerabilities exist.

### Additional environment variables

In case you want to add extra environment variables (useful for advanced operations like custom init scripts), you can use the `extraEnvVars` property.

```yaml
agent:
  extraEnvVars:
    - name: LOG_LEVEL
      value: error
```

Alternatively, you can use a ConfigMap or a Secret with the environment variables. To do so, use the `extraEnvVarsCM` or the `extraEnvVarsSecret` values.

### Sidecars

If additional containers are needed in the same pod as kubegems-local (such as additional metrics or logging exporters), they can be defined using the `sidecars` parameter. If these sidecars export extra ports, extra port definitions can be added using the `service.extraPorts` parameter.

### Pod affinity

This chart allows you to set your custom affinity using the `affinity` parameter. Find more information about Pod affinity in the [kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity).

As an alternative, use one of the preset configurations for pod affinity, pod anti-affinity, and node affinity available at the [bitnami/common](https://github.com/bitnami/charts/tree/master/bitnami/common#affinities) chart. To do so, set the `podAffinityPreset`, `podAntiAffinityPreset`, or `nodeAffinityPreset` parameters.

## Troubleshooting

Find more information about how to deal with common errors related to Bitnami's Helm charts in [this troubleshooting guide](https://docs.bitnami.com/general/how-to/troubleshoot-helm-chart-issues).

## License

Copyright &copy; 2022 Kubegems

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
