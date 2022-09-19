# kubegems

kubegems core components

## TL;DR

```console
$ helm repo add kubegems https://charts.kubegems.io/kubegems
$ helm install my-release kubegems/kubegems
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
helm install my-release kubegems/kubegemss
```

The command deploys kubegems on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

### Global parameters

| Name                      | Description                                     | Value  |
| ------------------------- | ----------------------------------------------- | ------ |
| `global.imageRegistry`    | Global Docker image registry                    | `""`   |
| `global.imagePullSecrets` | Global Docker registry secret names as an array | `[]`   |
| `global.storageClass`     | Global StorageClass for Persistent Volume(s)    | `""`   |
| `global.kubegemsVersion`  | Global kubegems version                         | `main` |


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


### dashboard Parameters

| Name                                              | Description                                                                                         | Value                |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------- | -------------------- |
| `dashboard.image.registry`                        | dashboard image registry                                                                            | `docker.io`          |
| `dashboard.image.repository`                      | dashboard image repository                                                                          | `kubegems/dashboard` |
| `dashboard.image.tag`                             | dashboard image tag (immutable tags are recommended)                                                | `latest`             |
| `dashboard.image.pullPolicy`                      | dashboard image pull policy                                                                         | `IfNotPresent`       |
| `dashboard.image.pullSecrets`                     | dashboard image pull secrets                                                                        | `[]`                 |
| `dashboard.image.debug`                           | Enable dashboard image debug mode                                                                   | `false`              |
| `dashboard.replicaCount`                          | Number of dashboard replicas to deploy                                                              | `1`                  |
| `dashboard.containerPorts.http`                   | dashboard HTTP container port                                                                       | `8000`               |
| `dashboard.livenessProbe.enabled`                 | Enable livenessProbe on dashboard containers                                                        | `true`               |
| `dashboard.livenessProbe.initialDelaySeconds`     | Initial delay seconds for livenessProbe                                                             | `10`                 |
| `dashboard.livenessProbe.periodSeconds`           | Period seconds for livenessProbe                                                                    | `20`                 |
| `dashboard.livenessProbe.timeoutSeconds`          | Timeout seconds for livenessProbe                                                                   | `1`                  |
| `dashboard.livenessProbe.failureThreshold`        | Failure threshold for livenessProbe                                                                 | `6`                  |
| `dashboard.livenessProbe.successThreshold`        | Success threshold for livenessProbe                                                                 | `1`                  |
| `dashboard.readinessProbe.enabled`                | Enable readinessProbe on dashboard containers                                                       | `true`               |
| `dashboard.readinessProbe.initialDelaySeconds`    | Initial delay seconds for readinessProbe                                                            | `10`                 |
| `dashboard.readinessProbe.periodSeconds`          | Period seconds for readinessProbe                                                                   | `20`                 |
| `dashboard.readinessProbe.timeoutSeconds`         | Timeout seconds for readinessProbe                                                                  | `1`                  |
| `dashboard.readinessProbe.failureThreshold`       | Failure threshold for readinessProbe                                                                | `6`                  |
| `dashboard.readinessProbe.successThreshold`       | Success threshold for readinessProbe                                                                | `1`                  |
| `dashboard.startupProbe.enabled`                  | Enable startupProbe on dashboard containers                                                         | `false`              |
| `dashboard.startupProbe.initialDelaySeconds`      | Initial delay seconds for startupProbe                                                              | `10`                 |
| `dashboard.startupProbe.periodSeconds`            | Period seconds for startupProbe                                                                     | `20`                 |
| `dashboard.startupProbe.timeoutSeconds`           | Timeout seconds for startupProbe                                                                    | `1`                  |
| `dashboard.startupProbe.failureThreshold`         | Failure threshold for startupProbe                                                                  | `6`                  |
| `dashboard.startupProbe.successThreshold`         | Success threshold for startupProbe                                                                  | `1`                  |
| `dashboard.customLivenessProbe`                   | Custom livenessProbe that overrides the default one                                                 | `{}`                 |
| `dashboard.customReadinessProbe`                  | Custom readinessProbe that overrides the default one                                                | `{}`                 |
| `dashboard.customStartupProbe`                    | Custom startupProbe that overrides the default one                                                  | `{}`                 |
| `dashboard.resources.limits`                      | The resources limits for the dashboard containers                                                   | `{}`                 |
| `dashboard.resources.requests`                    | The requested resources for the dashboard containers                                                | `{}`                 |
| `dashboard.podSecurityContext.enabled`            | Enabled dashboard pods' Security Context                                                            | `false`              |
| `dashboard.podSecurityContext.fsGroup`            | Set dashboard pod's Security Context fsGroup                                                        | `1001`               |
| `dashboard.containerSecurityContext.enabled`      | Enabled dashboard containers' Security Context                                                      | `false`              |
| `dashboard.containerSecurityContext.runAsUser`    | Set dashboard containers' Security Context runAsUser                                                | `1001`               |
| `dashboard.containerSecurityContext.runAsNonRoot` | Set dashboard containers' Security Context runAsNonRoot                                             | `true`               |
| `dashboard.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for dashboard                      | `""`                 |
| `dashboard.command`                               | Override default container command (useful when using custom images)                                | `[]`                 |
| `dashboard.args`                                  | Override default container args (useful when using custom images)                                   | `[]`                 |
| `dashboard.hostAliases`                           | dashboard pods host aliases                                                                         | `[]`                 |
| `dashboard.podLabels`                             | Extra labels for dashboard pods                                                                     | `{}`                 |
| `dashboard.podAnnotations`                        | Annotations for dashboard pods                                                                      | `{}`                 |
| `dashboard.podAffinityPreset`                     | Pod affinity preset. Ignored if `dashboard.affinity` is set. Allowed values: `soft` or `hard`       | `""`                 |
| `dashboard.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `dashboard.affinity` is set. Allowed values: `soft` or `hard`  | `soft`               |
| `dashboard.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `dashboard.affinity` is set. Allowed values: `soft` or `hard` | `""`                 |
| `dashboard.nodeAffinityPreset.key`                | Node label key to match. Ignored if `dashboard.affinity` is set                                     | `""`                 |
| `dashboard.nodeAffinityPreset.values`             | Node label values to match. Ignored if `dashboard.affinity` is set                                  | `[]`                 |
| `dashboard.affinity`                              | Affinity for dashboard pods assignment                                                              | `{}`                 |
| `dashboard.nodeSelector`                          | Node labels for dashboard pods assignment                                                           | `{}`                 |
| `dashboard.tolerations`                           | Tolerations for dashboard pods assignment                                                           | `[]`                 |
| `dashboard.updateStrategy.type`                   | dashboard statefulset strategy type                                                                 | `RollingUpdate`      |
| `dashboard.priorityClassName`                     | dashboard pods' priorityClassName                                                                   | `""`                 |
| `dashboard.schedulerName`                         | Name of the k8s scheduler (other than default) for dashboard pods                                   | `""`                 |
| `dashboard.lifecycleHooks`                        | for the dashboard container(s) to automate configuration before or after startup                    | `{}`                 |
| `dashboard.extraEnvVars`                          | Array with extra environment variables to add to dashboard nodes                                    | `[]`                 |
| `dashboard.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for dashboard nodes                            | `{}`                 |
| `dashboard.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for dashboard nodes                               | `{}`                 |
| `dashboard.extraVolumes`                          | Optionally specify extra list of additional volumes for the dashboard pod(s)                        | `[]`                 |
| `dashboard.extraVolumeMounts`                     | Optionally specify extra list of additional volumeMounts for the dashboard container(s)             | `[]`                 |
| `dashboard.sidecars`                              | Add additional sidecar containers to the dashboard pod(s)                                           | `{}`                 |
| `dashboard.initContainers`                        | Add additional init containers to the dashboard pod(s)                                              | `{}`                 |


### dashboard Exposure Parameters

| Name                                         | Description                                                                          | Value       |
| -------------------------------------------- | ------------------------------------------------------------------------------------ | ----------- |
| `dashboard.service.type`                     | dashboard service type                                                               | `ClusterIP` |
| `dashboard.service.ports.http`               | dashboard service HTTP port                                                          | `80`        |
| `dashboard.service.nodePorts.http`           | Node port for HTTP                                                                   | `""`        |
| `dashboard.service.clusterIP`                | dashboard service Cluster IP                                                         | `""`        |
| `dashboard.service.loadBalancerIP`           | dashboard service Load Balancer IP                                                   | `""`        |
| `dashboard.service.loadBalancerSourceRanges` | dashboard service Load Balancer sources                                              | `[]`        |
| `dashboard.service.externalTrafficPolicy`    | dashboard service external traffic policy                                            | `Cluster`   |
| `dashboard.service.annotations`              | Additional custom annotations for dashboard service                                  | `{}`        |
| `dashboard.service.extraPorts`               | Extra ports to expose in dashboard service (normally used with the `sidecars` value) | `[]`        |


### dashboard Metrics parameters

| Name                                                 | Description                                                                 | Value                    |
| ---------------------------------------------------- | --------------------------------------------------------------------------- | ------------------------ |
| `dashboard.metrics.enabled`                          | Create a service for accessing the metrics endpoint                         | `true`                   |
| `dashboard.metrics.service.type`                     | controller metrics service type                                             | `ClusterIP`              |
| `dashboard.metrics.service.port`                     | controller metrics service HTTP port                                        | `9100`                   |
| `dashboard.metrics.service.nodePort`                 | Node port for HTTP                                                          | `""`                     |
| `dashboard.metrics.service.clusterIP`                | controller metrics service Cluster IP                                       | `""`                     |
| `dashboard.metrics.service.extraPorts`               | Extra ports to expose (normally used with the `sidecar` value)              | `[]`                     |
| `dashboard.metrics.service.loadBalancerIP`           | controller metrics service Load Balancer IP                                 | `""`                     |
| `dashboard.metrics.service.loadBalancerSourceRanges` | controller metrics service Load Balancer sources                            | `[]`                     |
| `dashboard.metrics.service.externalTrafficPolicy`    | controller metrics service external traffic policy                          | `Cluster`                |
| `dashboard.metrics.service.annotations`              | Additional custom annotations for controller metrics service                | `{}`                     |
| `dashboard.metrics.serviceMonitor.enabled`           | Specify if a servicemonitor will be deployed for prometheus-operator        | `true`                   |
| `dashboard.metrics.serviceMonitor.jobLabel`          | Specify the jobLabel to use for the prometheus-operator                     | `app.kubernetes.io/name` |
| `dashboard.metrics.serviceMonitor.honorLabels`       | Honor metrics labels                                                        | `false`                  |
| `dashboard.metrics.serviceMonitor.selector`          | Prometheus instance selector labels                                         | `{}`                     |
| `dashboard.metrics.serviceMonitor.scrapeTimeout`     | Timeout after which the scrape is ended                                     | `""`                     |
| `dashboard.metrics.serviceMonitor.interval`          | Scrape interval. If not set, the Prometheus default scrape interval is used | `""`                     |
| `dashboard.metrics.serviceMonitor.additionalLabels`  | Used to pass Labels that are required by the installed Prometheus Operator  | `{}`                     |
| `dashboard.metrics.serviceMonitor.metricRelabelings` | Specify additional relabeling of metrics                                    | `[]`                     |
| `dashboard.metrics.serviceMonitor.relabelings`       | Specify general relabeling                                                  | `[]`                     |


### Init Job Parameters

| Name                                         | Description                                                                                    | Value                      |
| -------------------------------------------- | ---------------------------------------------------------------------------------------------- | -------------------------- |
| `init.charts.image.registry`                 | image registry                                                                                 | `docker.io`                |
| `init.charts.image.repository`               | image repository                                                                               | `kubegems/appstore-charts` |
| `init.charts.image.tag`                      | image tag (immutable tags are recommended)                                                     | `latest`                   |
| `init.charts.image.pullPolicy`               | image pull policy                                                                              | `IfNotPresent`             |
| `init.charts.image.pullSecrets`              | image pull secrets                                                                             | `[]`                       |
| `init.charts.image.debug`                    | Enable image debug mode                                                                        | `false`                    |
| `init.charts.replicaCount`                   | Number of API replicas to deploy                                                               | `1`                        |
| `init.charts.restartPolicy`                  | The restart policy for job,valid values: "OnFailure", "Never"                                  | `OnFailure`                |
| `init.charts.command`                        | Override default container command (useful when using custom images)                           | `[]`                       |
| `init.charts.args`                           | Override default container args (useful when using custom images)                              | `[]`                       |
| `init.image.registry`                        | API image registry                                                                             | `docker.io`                |
| `init.image.repository`                      | API image repository                                                                           | `kubegems/kubegems`        |
| `init.image.tag`                             | API image tag (immutable tags are recommended)                                                 | `latest`                   |
| `init.image.pullPolicy`                      | API image pull policy                                                                          | `IfNotPresent`             |
| `init.image.pullSecrets`                     | API image pull secrets                                                                         | `[]`                       |
| `init.image.debug`                           | Enable API image debug mode                                                                    | `false`                    |
| `init.replicaCount`                          | Number of API replicas to deploy                                                               | `1`                        |
| `init.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for API                       | `""`                       |
| `init.restartPolicy`                         | The restart policy for job,valid values: "OnFailure", "Never"                                  | `OnFailure`                |
| `init.command`                               | Override default container command (useful when using custom images)                           | `[]`                       |
| `init.args`                                  | Override default container args (useful when using custom images)                              | `[]`                       |
| `init.hostAliases`                           | API pods host aliases                                                                          | `[]`                       |
| `init.podLabels`                             | Extra labels for API pods                                                                      | `{}`                       |
| `init.podAnnotations`                        | Annotations for API pods                                                                       | `{}`                       |
| `init.podAffinityPreset`                     | Pod affinity preset. Ignored if `init.affinity` is set. Allowed values: `soft` or `hard`       | `""`                       |
| `init.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `init.affinity` is set. Allowed values: `soft` or `hard`  | `soft`                     |
| `init.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `init.affinity` is set. Allowed values: `soft` or `hard` | `""`                       |
| `init.nodeAffinityPreset.key`                | Node label key to match. Ignored if `init.affinity` is set                                     | `""`                       |
| `init.nodeAffinityPreset.values`             | Node label values to match. Ignored if `init.affinity` is set                                  | `[]`                       |
| `init.affinity`                              | Affinity for API pods assignment                                                               | `{}`                       |
| `init.nodeSelector`                          | Node labels for API pods assignment                                                            | `{}`                       |
| `init.tolerations`                           | Tolerations for API pods assignment                                                            | `[]`                       |
| `init.updateStrategy.type`                   | API statefulset strategy type                                                                  | `RollingUpdate`            |
| `init.priorityClassName`                     | API pods' priorityClassName                                                                    | `""`                       |
| `init.schedulerName`                         | Name of the k8s scheduler (other than default) for API pods                                    | `""`                       |
| `init.lifecycleHooks`                        | for the API container(s) to automate configuration before or after startup                     | `{}`                       |
| `init.extraEnvVars`                          | Array with extra environment variables to add to API nodes                                     | `[]`                       |
| `init.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for API nodes                             | `{}`                       |
| `init.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for API nodes                                | `{}`                       |
| `init.resources.limits`                      | The resources limits for the API containers                                                    | `{}`                       |
| `init.resources.requests`                    | The requested resources for the API containers                                                 | `{}`                       |
| `init.podSecurityContext.enabled`            | Enabled API pods' Security Context                                                             | `false`                    |
| `init.podSecurityContext.fsGroup`            | Set API pod's Security Context fsGroup                                                         | `1001`                     |
| `init.containerSecurityContext.enabled`      | Enabled API containers' Security Context                                                       | `false`                    |
| `init.containerSecurityContext.runAsUser`    | Set API containers' Security Context runAsUser                                                 | `1001`                     |
| `init.containerSecurityContext.runAsNonRoot` | Set API containers' Security Context runAsNonRoot                                              | `true`                     |


### API Parameters

| Name                                        | Description                                                                                   | Value               |
| ------------------------------------------- | --------------------------------------------------------------------------------------------- | ------------------- |
| `api.image.registry`                        | API image registry                                                                            | `docker.io`         |
| `api.image.repository`                      | API image repository                                                                          | `kubegems/kubegems` |
| `api.image.tag`                             | API image tag (immutable tags are recommended)                                                | `latest`            |
| `api.image.pullPolicy`                      | API image pull policy                                                                         | `IfNotPresent`      |
| `api.image.pullSecrets`                     | API image pull secrets                                                                        | `[]`                |
| `api.image.debug`                           | Enable API image debug mode                                                                   | `false`             |
| `api.replicaCount`                          | Number of API replicas to deploy                                                              | `1`                 |
| `api.containerPorts.http`                   | API HTTP container port                                                                       | `8080`              |
| `api.livenessProbe.enabled`                 | Enable livenessProbe on API containers                                                        | `true`              |
| `api.livenessProbe.initialDelaySeconds`     | Initial delay seconds for livenessProbe                                                       | `10`                |
| `api.livenessProbe.periodSeconds`           | Period seconds for livenessProbe                                                              | `20`                |
| `api.livenessProbe.timeoutSeconds`          | Timeout seconds for livenessProbe                                                             | `1`                 |
| `api.livenessProbe.failureThreshold`        | Failure threshold for livenessProbe                                                           | `6`                 |
| `api.livenessProbe.successThreshold`        | Success threshold for livenessProbe                                                           | `1`                 |
| `api.readinessProbe.enabled`                | Enable readinessProbe on API containers                                                       | `true`              |
| `api.readinessProbe.initialDelaySeconds`    | Initial delay seconds for readinessProbe                                                      | `10`                |
| `api.readinessProbe.periodSeconds`          | Period seconds for readinessProbe                                                             | `20`                |
| `api.readinessProbe.timeoutSeconds`         | Timeout seconds for readinessProbe                                                            | `1`                 |
| `api.readinessProbe.failureThreshold`       | Failure threshold for readinessProbe                                                          | `6`                 |
| `api.readinessProbe.successThreshold`       | Success threshold for readinessProbe                                                          | `1`                 |
| `api.startupProbe.enabled`                  | Enable startupProbe on API containers                                                         | `false`             |
| `api.startupProbe.initialDelaySeconds`      | Initial delay seconds for startupProbe                                                        | `10`                |
| `api.startupProbe.periodSeconds`            | Period seconds for startupProbe                                                               | `20`                |
| `api.startupProbe.timeoutSeconds`           | Timeout seconds for startupProbe                                                              | `1`                 |
| `api.startupProbe.failureThreshold`         | Failure threshold for startupProbe                                                            | `6`                 |
| `api.startupProbe.successThreshold`         | Success threshold for startupProbe                                                            | `1`                 |
| `api.customLivenessProbe`                   | Custom livenessProbe that overrides the default one                                           | `{}`                |
| `api.customReadinessProbe`                  | Custom readinessProbe that overrides the default one                                          | `{}`                |
| `api.customStartupProbe`                    | Custom startupProbe that overrides the default one                                            | `{}`                |
| `api.resources.limits`                      | The resources limits for the API containers                                                   | `{}`                |
| `api.resources.requests`                    | The requested resources for the API containers                                                | `{}`                |
| `api.podSecurityContext.enabled`            | Enabled API pods' Security Context                                                            | `false`             |
| `api.podSecurityContext.fsGroup`            | Set API pod's Security Context fsGroup                                                        | `1001`              |
| `api.containerSecurityContext.enabled`      | Enabled API containers' Security Context                                                      | `false`             |
| `api.containerSecurityContext.runAsUser`    | Set API containers' Security Context runAsUser                                                | `1001`              |
| `api.containerSecurityContext.runAsNonRoot` | Set API containers' Security Context runAsNonRoot                                             | `true`              |
| `api.jwt.enabled`                           | Enable jwt authentication                                                                     | `true`              |
| `api.jwt.useCertManager`                    | using cert-manager for jwt secret generation                                                  | `false`             |
| `api.jwt.secretName`                        | secret name alternative                                                                       | `""`                |
| `api.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for API                      | `""`                |
| `api.command`                               | Override default container command (useful when using custom images)                          | `[]`                |
| `api.args`                                  | Override default container args (useful when using custom images)                             | `[]`                |
| `api.hostAliases`                           | API pods host aliases                                                                         | `[]`                |
| `api.podLabels`                             | Extra labels for API pods                                                                     | `{}`                |
| `api.podAnnotations`                        | Annotations for API pods                                                                      | `{}`                |
| `api.podAffinityPreset`                     | Pod affinity preset. Ignored if `api.affinity` is set. Allowed values: `soft` or `hard`       | `""`                |
| `api.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `api.affinity` is set. Allowed values: `soft` or `hard`  | `soft`              |
| `api.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `api.affinity` is set. Allowed values: `soft` or `hard` | `""`                |
| `api.nodeAffinityPreset.key`                | Node label key to match. Ignored if `api.affinity` is set                                     | `""`                |
| `api.nodeAffinityPreset.values`             | Node label values to match. Ignored if `api.affinity` is set                                  | `[]`                |
| `api.affinity`                              | Affinity for API pods assignment                                                              | `{}`                |
| `api.nodeSelector`                          | Node labels for API pods assignment                                                           | `{}`                |
| `api.tolerations`                           | Tolerations for API pods assignment                                                           | `[]`                |
| `api.updateStrategy.type`                   | API statefulset strategy type                                                                 | `RollingUpdate`     |
| `api.priorityClassName`                     | API pods' priorityClassName                                                                   | `""`                |
| `api.schedulerName`                         | Name of the k8s scheduler (other than default) for API pods                                   | `""`                |
| `api.lifecycleHooks`                        | for the API container(s) to automate configuration before or after startup                    | `{}`                |
| `api.extraEnvVars`                          | Array with extra environment variables to add to API nodes                                    | `[]`                |
| `api.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for API nodes                            | `{}`                |
| `api.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for API nodes                               | `{}`                |
| `api.extraVolumes`                          | Optionally specify extra list of additional volumes for the API pod(s)                        | `[]`                |
| `api.extraVolumeMounts`                     | Optionally specify extra list of additional volumeMounts for the API container(s)             | `[]`                |
| `api.sidecars`                              | Add additional sidecar containers to the API pod(s)                                           | `{}`                |
| `api.initContainers`                        | Add additional init containers to the API pod(s)                                              | `{}`                |


### API Exposure Parameters

| Name                                   | Description                                                                    | Value       |
| -------------------------------------- | ------------------------------------------------------------------------------ | ----------- |
| `api.service.type`                     | API service type                                                               | `ClusterIP` |
| `api.service.ports.http`               | API service HTTP port                                                          | `80`        |
| `api.service.nodePorts.http`           | Node port for HTTP                                                             | `0`         |
| `api.service.clusterIP`                | API service Cluster IP                                                         | `""`        |
| `api.service.loadBalancerIP`           | API service Load Balancer IP                                                   | `""`        |
| `api.service.loadBalancerSourceRanges` | API service Load Balancer sources                                              | `[]`        |
| `api.service.externalTrafficPolicy`    | API service external traffic policy                                            | `Cluster`   |
| `api.service.annotations`              | Additional custom annotations for API service                                  | `{}`        |
| `api.service.extraPorts`               | Extra ports to expose in API service (normally used with the `sidecars` value) | `[]`        |


### API Metrics parameters

| Name                                           | Description                                                                 | Value                    |
| ---------------------------------------------- | --------------------------------------------------------------------------- | ------------------------ |
| `api.metrics.enabled`                          | Create a service for accessing the metrics endpoint                         | `true`                   |
| `api.metrics.service.type`                     | controller metrics service type                                             | `ClusterIP`              |
| `api.metrics.service.port`                     | controller metrics service HTTP port                                        | `9100`                   |
| `api.metrics.service.nodePort`                 | Node port for HTTP                                                          | `""`                     |
| `api.metrics.service.clusterIP`                | controller metrics service Cluster IP                                       | `""`                     |
| `api.metrics.service.extraPorts`               | Extra ports to expose (normally used with the `sidecar` value)              | `[]`                     |
| `api.metrics.service.loadBalancerIP`           | controller metrics service Load Balancer IP                                 | `""`                     |
| `api.metrics.service.loadBalancerSourceRanges` | controller metrics service Load Balancer sources                            | `[]`                     |
| `api.metrics.service.externalTrafficPolicy`    | controller metrics service external traffic policy                          | `Cluster`                |
| `api.metrics.service.annotations`              | Additional custom annotations for controller metrics service                | `{}`                     |
| `api.metrics.serviceMonitor.enabled`           | Specify if a servicemonitor will be deployed for prometheus-operator        | `true`                   |
| `api.metrics.serviceMonitor.jobLabel`          | Specify the jobLabel to use for the prometheus-operator                     | `app.kubernetes.io/name` |
| `api.metrics.serviceMonitor.honorLabels`       | Honor metrics labels                                                        | `false`                  |
| `api.metrics.serviceMonitor.selector`          | Prometheus instance selector labels                                         | `{}`                     |
| `api.metrics.serviceMonitor.scrapeTimeout`     | Timeout after which the scrape is ended                                     | `""`                     |
| `api.metrics.serviceMonitor.interval`          | Scrape interval. If not set, the Prometheus default scrape interval is used | `""`                     |
| `api.metrics.serviceMonitor.additionalLabels`  | Used to pass Labels that are required by the installed Prometheus Operator  | `{}`                     |
| `api.metrics.serviceMonitor.metricRelabelings` | Specify additional relabeling of metrics                                    | `[]`                     |
| `api.metrics.serviceMonitor.relabelings`       | Specify general relabeling                                                  | `[]`                     |


### msgbus Parameters

| Name                                           | Description                                                                                      | Value               |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------- |
| `msgbus.image.registry`                        | msgbus image registry                                                                            | `docker.io`         |
| `msgbus.image.repository`                      | msgbus image repository                                                                          | `kubegems/kubegems` |
| `msgbus.image.tag`                             | msgbus image tag (immutable tags are recommended)                                                | `latest`            |
| `msgbus.image.pullPolicy`                      | msgbus image pull policy                                                                         | `IfNotPresent`      |
| `msgbus.image.pullSecrets`                     | msgbus image pull secrets                                                                        | `[]`                |
| `msgbus.image.debug`                           | Enable msgbus image debug mode                                                                   | `false`             |
| `msgbus.replicaCount`                          | Number of msgbus replicas to deploy                                                              | `1`                 |
| `msgbus.containerPorts.http`                   | msgbus HTTP container port                                                                       | `8080`              |
| `msgbus.livenessProbe.enabled`                 | Enable livenessProbe on msgbus containers                                                        | `true`              |
| `msgbus.livenessProbe.initialDelaySeconds`     | Initial delay seconds for livenessProbe                                                          | `10`                |
| `msgbus.livenessProbe.periodSeconds`           | Period seconds for livenessProbe                                                                 | `20`                |
| `msgbus.livenessProbe.timeoutSeconds`          | Timeout seconds for livenessProbe                                                                | `1`                 |
| `msgbus.livenessProbe.failureThreshold`        | Failure threshold for livenessProbe                                                              | `6`                 |
| `msgbus.livenessProbe.successThreshold`        | Success threshold for livenessProbe                                                              | `1`                 |
| `msgbus.readinessProbe.enabled`                | Enable readinessProbe on msgbus containers                                                       | `true`              |
| `msgbus.readinessProbe.initialDelaySeconds`    | Initial delay seconds for readinessProbe                                                         | `10`                |
| `msgbus.readinessProbe.periodSeconds`          | Period seconds for readinessProbe                                                                | `20`                |
| `msgbus.readinessProbe.timeoutSeconds`         | Timeout seconds for readinessProbe                                                               | `1`                 |
| `msgbus.readinessProbe.failureThreshold`       | Failure threshold for readinessProbe                                                             | `6`                 |
| `msgbus.readinessProbe.successThreshold`       | Success threshold for readinessProbe                                                             | `1`                 |
| `msgbus.startupProbe.enabled`                  | Enable startupProbe on msgbus containers                                                         | `false`             |
| `msgbus.startupProbe.initialDelaySeconds`      | Initial delay seconds for startupProbe                                                           | `10`                |
| `msgbus.startupProbe.periodSeconds`            | Period seconds for startupProbe                                                                  | `20`                |
| `msgbus.startupProbe.timeoutSeconds`           | Timeout seconds for startupProbe                                                                 | `1`                 |
| `msgbus.startupProbe.failureThreshold`         | Failure threshold for startupProbe                                                               | `6`                 |
| `msgbus.startupProbe.successThreshold`         | Success threshold for startupProbe                                                               | `1`                 |
| `msgbus.customLivenessProbe`                   | Custom livenessProbe that overrides the default one                                              | `{}`                |
| `msgbus.customReadinessProbe`                  | Custom readinessProbe that overrides the default one                                             | `{}`                |
| `msgbus.customStartupProbe`                    | Custom startupProbe that overrides the default one                                               | `{}`                |
| `msgbus.resources.limits`                      | The resources limits for the msgbus containers                                                   | `{}`                |
| `msgbus.resources.requests`                    | The requested resources for the msgbus containers                                                | `{}`                |
| `msgbus.podSecurityContext.enabled`            | Enabled msgbus pods' Security Context                                                            | `false`             |
| `msgbus.podSecurityContext.fsGroup`            | Set msgbus pod's Security Context fsGroup                                                        | `1001`              |
| `msgbus.containerSecurityContext.enabled`      | Enabled msgbus containers' Security Context                                                      | `false`             |
| `msgbus.containerSecurityContext.runAsUser`    | Set msgbus containers' Security Context runAsUser                                                | `1001`              |
| `msgbus.containerSecurityContext.runAsNonRoot` | Set msgbus containers' Security Context runAsNonRoot                                             | `true`              |
| `msgbus.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for msgbus                      | `""`                |
| `msgbus.command`                               | Override default container command (useful when using custom images)                             | `[]`                |
| `msgbus.args`                                  | Override default container args (useful when using custom images)                                | `[]`                |
| `msgbus.hostAliases`                           | msgbus pods host aliases                                                                         | `[]`                |
| `msgbus.podLabels`                             | Extra labels for msgbus pods                                                                     | `{}`                |
| `msgbus.podAnnotations`                        | Annotations for msgbus pods                                                                      | `{}`                |
| `msgbus.podAffinityPreset`                     | Pod affinity preset. Ignored if `msgbus.affinity` is set. Allowed values: `soft` or `hard`       | `""`                |
| `msgbus.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `msgbus.affinity` is set. Allowed values: `soft` or `hard`  | `soft`              |
| `msgbus.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `msgbus.affinity` is set. Allowed values: `soft` or `hard` | `""`                |
| `msgbus.nodeAffinityPreset.key`                | Node label key to match. Ignored if `msgbus.affinity` is set                                     | `""`                |
| `msgbus.nodeAffinityPreset.values`             | Node label values to match. Ignored if `msgbus.affinity` is set                                  | `[]`                |
| `msgbus.affinity`                              | Affinity for msgbus pods assignment                                                              | `{}`                |
| `msgbus.nodeSelector`                          | Node labels for msgbus pods assignment                                                           | `{}`                |
| `msgbus.tolerations`                           | Tolerations for msgbus pods assignment                                                           | `[]`                |
| `msgbus.updateStrategy.type`                   | msgbus statefulset strategy type                                                                 | `RollingUpdate`     |
| `msgbus.priorityClassName`                     | msgbus pods' priorityClassName                                                                   | `""`                |
| `msgbus.schedulerName`                         | Name of the k8s scheduler (other than default) for msgbus pods                                   | `""`                |
| `msgbus.lifecycleHooks`                        | for the msgbus container(s) to automate configuration before or after startup                    | `{}`                |
| `msgbus.extraEnvVars`                          | Array with extra environment variables to add to msgbus nodes                                    | `[]`                |
| `msgbus.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for msgbus nodes                            | `{}`                |
| `msgbus.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for msgbus nodes                               | `{}`                |
| `msgbus.extraVolumes`                          | Optionally specify extra list of additional volumes for the msgbus pod(s)                        | `[]`                |
| `msgbus.extraVolumeMounts`                     | Optionally specify extra list of additional volumeMounts for the msgbus container(s)             | `[]`                |
| `msgbus.sidecars`                              | Add additional sidecar containers to the msgbus pod(s)                                           | `{}`                |
| `msgbus.initContainers`                        | Add additional init containers to the msgbus pod(s)                                              | `{}`                |


### msgbus Exposure Parameters

| Name                                      | Description                                                                       | Value       |
| ----------------------------------------- | --------------------------------------------------------------------------------- | ----------- |
| `msgbus.service.type`                     | msgbus service type                                                               | `ClusterIP` |
| `msgbus.service.ports.http`               | msgbus service HTTP port                                                          | `80`        |
| `msgbus.service.nodePorts.http`           | Node port for HTTP                                                                | `""`        |
| `msgbus.service.clusterIP`                | msgbus service Cluster IP                                                         | `""`        |
| `msgbus.service.loadBalancerIP`           | msgbus service Load Balancer IP                                                   | `""`        |
| `msgbus.service.loadBalancerSourceRanges` | msgbus service Load Balancer sources                                              | `[]`        |
| `msgbus.service.externalTrafficPolicy`    | msgbus service external traffic policy                                            | `Cluster`   |
| `msgbus.service.annotations`              | Additional custom annotations for msgbus service                                  | `{}`        |
| `msgbus.service.extraPorts`               | Extra ports to expose in msgbus service (normally used with the `sidecars` value) | `[]`        |


### msgbus Metrics parameters

| Name                                              | Description                                                                 | Value                    |
| ------------------------------------------------- | --------------------------------------------------------------------------- | ------------------------ |
| `msgbus.metrics.enabled`                          | Create a service for accessing the metrics endpoint                         | `true`                   |
| `msgbus.metrics.service.type`                     | controller metrics service type                                             | `ClusterIP`              |
| `msgbus.metrics.service.port`                     | controller metrics service HTTP port                                        | `9100`                   |
| `msgbus.metrics.service.nodePort`                 | Node port for HTTP                                                          | `""`                     |
| `msgbus.metrics.service.clusterIP`                | controller metrics service Cluster IP                                       | `""`                     |
| `msgbus.metrics.service.extraPorts`               | Extra ports to expose (normally used with the `sidecar` value)              | `[]`                     |
| `msgbus.metrics.service.loadBalancerIP`           | controller metrics service Load Balancer IP                                 | `""`                     |
| `msgbus.metrics.service.loadBalancerSourceRanges` | controller metrics service Load Balancer sources                            | `[]`                     |
| `msgbus.metrics.service.externalTrafficPolicy`    | controller metrics service external traffic policy                          | `Cluster`                |
| `msgbus.metrics.service.annotations`              | Additional custom annotations for controller metrics service                | `{}`                     |
| `msgbus.metrics.serviceMonitor.enabled`           | Specify if a servicemonitor will be deployed for prometheus-operator        | `true`                   |
| `msgbus.metrics.serviceMonitor.jobLabel`          | Specify the jobLabel to use for the prometheus-operator                     | `app.kubernetes.io/name` |
| `msgbus.metrics.serviceMonitor.honorLabels`       | Honor metrics labels                                                        | `false`                  |
| `msgbus.metrics.serviceMonitor.selector`          | Prometheus instance selector labels                                         | `{}`                     |
| `msgbus.metrics.serviceMonitor.scrapeTimeout`     | Timeout after which the scrape is ended                                     | `""`                     |
| `msgbus.metrics.serviceMonitor.interval`          | Scrape interval. If not set, the Prometheus default scrape interval is used | `""`                     |
| `msgbus.metrics.serviceMonitor.additionalLabels`  | Used to pass Labels that are required by the installed Prometheus Operator  | `{}`                     |
| `msgbus.metrics.serviceMonitor.metricRelabelings` | Specify additional relabeling of metrics                                    | `[]`                     |
| `msgbus.metrics.serviceMonitor.relabelings`       | Specify general relabeling                                                  | `[]`                     |


### Worker Parameters

| Name                                           | Description                                                                                      | Value               |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------- |
| `worker.image.registry`                        | worker image registry                                                                            | `docker.io`         |
| `worker.image.repository`                      | worker image repository                                                                          | `kubegems/kubegems` |
| `worker.image.tag`                             | worker image tag (immutable tags are recommended)                                                | `latest`            |
| `worker.image.pullPolicy`                      | worker image pull policy                                                                         | `IfNotPresent`      |
| `worker.image.pullSecrets`                     | worker image pull secrets                                                                        | `[]`                |
| `worker.image.debug`                           | Enable worker image debug mode                                                                   | `false`             |
| `worker.replicaCount`                          | Number of worker replicas to deploy                                                              | `1`                 |
| `worker.containerPorts.http`                   | worker HTTP container port                                                                       | `8080`              |
| `worker.livenessProbe.enabled`                 | Enable livenessProbe on worker containers                                                        | `false`             |
| `worker.livenessProbe.initialDelaySeconds`     | Initial delay seconds for livenessProbe                                                          | `10`                |
| `worker.livenessProbe.periodSeconds`           | Period seconds for livenessProbe                                                                 | `20`                |
| `worker.livenessProbe.timeoutSeconds`          | Timeout seconds for livenessProbe                                                                | `1`                 |
| `worker.livenessProbe.failureThreshold`        | Failure threshold for livenessProbe                                                              | `6`                 |
| `worker.livenessProbe.successThreshold`        | Success threshold for livenessProbe                                                              | `1`                 |
| `worker.readinessProbe.enabled`                | Enable readinessProbe on worker containers                                                       | `false`             |
| `worker.readinessProbe.initialDelaySeconds`    | Initial delay seconds for readinessProbe                                                         | `10`                |
| `worker.readinessProbe.periodSeconds`          | Period seconds for readinessProbe                                                                | `20`                |
| `worker.readinessProbe.timeoutSeconds`         | Timeout seconds for readinessProbe                                                               | `1`                 |
| `worker.readinessProbe.failureThreshold`       | Failure threshold for readinessProbe                                                             | `6`                 |
| `worker.readinessProbe.successThreshold`       | Success threshold for readinessProbe                                                             | `1`                 |
| `worker.startupProbe.enabled`                  | Enable startupProbe on worker containers                                                         | `false`             |
| `worker.startupProbe.initialDelaySeconds`      | Initial delay seconds for startupProbe                                                           | `10`                |
| `worker.startupProbe.periodSeconds`            | Period seconds for startupProbe                                                                  | `20`                |
| `worker.startupProbe.timeoutSeconds`           | Timeout seconds for startupProbe                                                                 | `1`                 |
| `worker.startupProbe.failureThreshold`         | Failure threshold for startupProbe                                                               | `6`                 |
| `worker.startupProbe.successThreshold`         | Success threshold for startupProbe                                                               | `1`                 |
| `worker.customLivenessProbe`                   | Custom livenessProbe that overrides the default one                                              | `{}`                |
| `worker.customReadinessProbe`                  | Custom readinessProbe that overrides the default one                                             | `{}`                |
| `worker.customStartupProbe`                    | Custom startupProbe that overrides the default one                                               | `{}`                |
| `worker.resources.limits`                      | The resources limits for the worker containers                                                   | `{}`                |
| `worker.resources.requests`                    | The requested resources for the worker containers                                                | `{}`                |
| `worker.podSecurityContext.enabled`            | Enabled worker pods' Security Context                                                            | `false`             |
| `worker.podSecurityContext.fsGroup`            | Set worker pod's Security Context fsGroup                                                        | `1001`              |
| `worker.containerSecurityContext.enabled`      | Enabled worker containers' Security Context                                                      | `false`             |
| `worker.containerSecurityContext.runAsUser`    | Set worker containers' Security Context runAsUser                                                | `1001`              |
| `worker.containerSecurityContext.runAsNonRoot` | Set worker containers' Security Context runAsNonRoot                                             | `true`              |
| `worker.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for worker                      | `""`                |
| `worker.command`                               | Override default container command (useful when using custom images)                             | `[]`                |
| `worker.args`                                  | Override default container args (useful when using custom images)                                | `[]`                |
| `worker.hostAliases`                           | worker pods host aliases                                                                         | `[]`                |
| `worker.podLabels`                             | Extra labels for worker pods                                                                     | `{}`                |
| `worker.podAnnotations`                        | Annotations for worker pods                                                                      | `{}`                |
| `worker.podAffinityPreset`                     | Pod affinity preset. Ignored if `worker.affinity` is set. Allowed values: `soft` or `hard`       | `""`                |
| `worker.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `worker.affinity` is set. Allowed values: `soft` or `hard`  | `soft`              |
| `worker.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `worker.affinity` is set. Allowed values: `soft` or `hard` | `""`                |
| `worker.nodeAffinityPreset.key`                | Node label key to match. Ignored if `worker.affinity` is set                                     | `""`                |
| `worker.nodeAffinityPreset.values`             | Node label values to match. Ignored if `worker.affinity` is set                                  | `[]`                |
| `worker.affinity`                              | Affinity for worker pods assignment                                                              | `{}`                |
| `worker.nodeSelector`                          | Node labels for worker pods assignment                                                           | `{}`                |
| `worker.tolerations`                           | Tolerations for worker pods assignment                                                           | `[]`                |
| `worker.updateStrategy.type`                   | worker statefulset strategy type                                                                 | `RollingUpdate`     |
| `worker.priorityClassName`                     | worker pods' priorityClassName                                                                   | `""`                |
| `worker.schedulerName`                         | Name of the k8s scheduler (other than default) for worker pods                                   | `""`                |
| `worker.lifecycleHooks`                        | for the worker container(s) to automate configuration before or after startup                    | `{}`                |
| `worker.extraEnvVars`                          | Array with extra environment variables to add to worker nodes                                    | `[]`                |
| `worker.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for worker nodes                            | `{}`                |
| `worker.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for worker nodes                               | `{}`                |
| `worker.extraVolumes`                          | Optionally specify extra list of additional volumes for the worker pod(s)                        | `[]`                |
| `worker.extraVolumeMounts`                     | Optionally specify extra list of additional volumeMounts for the worker container(s)             | `[]`                |
| `worker.sidecars`                              | Add additional sidecar containers to the worker pod(s)                                           | `{}`                |
| `worker.initContainers`                        | Add additional init containers to the worker pod(s)                                              | `{}`                |


### worker Exposure Parameters

| Name                                      | Description                                                                       | Value       |
| ----------------------------------------- | --------------------------------------------------------------------------------- | ----------- |
| `worker.service.type`                     | worker service type                                                               | `ClusterIP` |
| `worker.service.ports.http`               | worker service HTTP port                                                          | `80`        |
| `worker.service.nodePorts.http`           | Node port for HTTP                                                                | `""`        |
| `worker.service.clusterIP`                | worker service Cluster IP                                                         | `""`        |
| `worker.service.loadBalancerIP`           | worker service Load Balancer IP                                                   | `""`        |
| `worker.service.loadBalancerSourceRanges` | worker service Load Balancer sources                                              | `[]`        |
| `worker.service.externalTrafficPolicy`    | worker service external traffic policy                                            | `Cluster`   |
| `worker.service.annotations`              | Additional custom annotations for worker service                                  | `{}`        |
| `worker.service.extraPorts`               | Extra ports to expose in worker service (normally used with the `sidecars` value) | `[]`        |


### worker Metrics parameters

| Name                                              | Description                                                                                                                      | Value                    |
| ------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------------------ |
| `worker.metrics.enabled`                          | Create a service for accessing the metrics endpoint                                                                              | `true`                   |
| `worker.metrics.service.type`                     | controller metrics service type                                                                                                  | `ClusterIP`              |
| `worker.metrics.service.port`                     | controller metrics service HTTP port                                                                                             | `9100`                   |
| `worker.metrics.service.nodePort`                 | Node port for HTTP                                                                                                               | `""`                     |
| `worker.metrics.service.clusterIP`                | controller metrics service Cluster IP                                                                                            | `""`                     |
| `worker.metrics.service.extraPorts`               | Extra ports to expose (normally used with the `sidecar` value)                                                                   | `[]`                     |
| `worker.metrics.service.loadBalancerIP`           | controller metrics service Load Balancer IP                                                                                      | `""`                     |
| `worker.metrics.service.loadBalancerSourceRanges` | controller metrics service Load Balancer sources                                                                                 | `[]`                     |
| `worker.metrics.service.externalTrafficPolicy`    | controller metrics service external traffic policy                                                                               | `Cluster`                |
| `worker.metrics.service.annotations`              | Additional custom annotations for controller metrics service                                                                     | `{}`                     |
| `worker.metrics.serviceMonitor.enabled`           | Specify if a servicemonitor will be deployed for prometheus-operator                                                             | `true`                   |
| `worker.metrics.serviceMonitor.jobLabel`          | Specify the jobLabel to use for the prometheus-operator                                                                          | `app.kubernetes.io/name` |
| `worker.metrics.serviceMonitor.honorLabels`       | Honor metrics labels                                                                                                             | `false`                  |
| `worker.metrics.serviceMonitor.selector`          | Prometheus instance selector labels                                                                                              | `{}`                     |
| `worker.metrics.serviceMonitor.scrapeTimeout`     | Timeout after which the scrape is ended                                                                                          | `""`                     |
| `worker.metrics.serviceMonitor.interval`          | Scrape interval. If not set, the Prometheus default scrape interval is used                                                      | `""`                     |
| `worker.metrics.serviceMonitor.additionalLabels`  | Used to pass Labels that are required by the installed Prometheus Operator                                                       | `{}`                     |
| `worker.metrics.serviceMonitor.metricRelabelings` | Specify additional relabeling of metrics                                                                                         | `[]`                     |
| `worker.metrics.serviceMonitor.relabelings`       | Specify general relabeling                                                                                                       | `[]`                     |
| `ingress.enabled`                                 | Enable ingress record generation for API                                                                                         | `true`                   |
| `ingress.pathType`                                | Ingress path type                                                                                                                | `ImplementationSpecific` |
| `ingress.apiVersion`                              | Force Ingress API version (automatically detected if not set)                                                                    | `""`                     |
| `ingress.hostname`                                | Default host for the ingress record                                                                                              | `dashboard.kubegems.io`  |
| `ingress.ingressClassName`                        | Ingress class name                                                                                                               | `default-gateway`        |
| `ingress.path`                                    | Default path for the ingress record                                                                                              | `/`                      |
| `ingress.annotations`                             | Additional annotations for the Ingress resource. To enable certificate autogeneration, place here your cert-manager annotations. | `{}`                     |
| `ingress.tls`                                     | Enable TLS configuration for the host defined at `ingress.hostname` parameter                                                    | `false`                  |
| `ingress.selfSigned`                              | Create a TLS secret for this ingress record using self-signed certificates generated by Helm                                     | `false`                  |
| `ingress.extraHosts`                              | An array with additional hostname(s) to be covered with the ingress record                                                       | `[]`                     |
| `ingress.extraPaths`                              | An array with additional arbitrary paths that may need to be added to the ingress under the main host                            | `[]`                     |
| `ingress.extraTls`                                | TLS configuration for additional hostname(s) to be covered with this ingress record                                              | `[]`                     |
| `ingress.secrets`                                 | Custom TLS certificates as secrets                                                                                               | `[]`                     |


### Init Container Parameters

| Name                                                   | Description                                                                                     | Value                              |
| ------------------------------------------------------ | ----------------------------------------------------------------------------------------------- | ---------------------------------- |
| `volumePermissions.enabled`                            | Enable init container that changes the owner/group of the PV mount point to `runAsUser:fsGroup` | `false`                            |
| `volumePermissions.image.registry`                     | Bitnami Shell image registry                                                                    | `docker.io`                        |
| `volumePermissions.image.repository`                   | Bitnami Shell image repository                                                                  | `bitnami/bitnami-shell`            |
| `volumePermissions.image.tag`                          | Bitnami Shell image tag (immutable tags are recommended)                                        | `10-debian-10-r%%IMAGE_REVISION%%` |
| `volumePermissions.image.pullPolicy`                   | Bitnami Shell image pull policy                                                                 | `IfNotPresent`                     |
| `volumePermissions.image.pullSecrets`                  | Bitnami Shell image pull secrets                                                                | `[]`                               |
| `volumePermissions.resources.limits`                   | The resources limits for the init container                                                     | `{}`                               |
| `volumePermissions.resources.requests`                 | The requested resources for the init container                                                  | `{}`                               |
| `volumePermissions.containerSecurityContext.runAsUser` | Set init container's Security Context runAsUser                                                 | `0`                                |


### Persistence parameters

| Name                        | Description                           | Value               |
| --------------------------- | ------------------------------------- | ------------------- |
| `persistence.enabled`       | Enable persistent storage             | `true`              |
| `persistence.storageClass`  | Storage class name                    | `""`                |
| `persistence.accessModes`   | PVC Access Mode for volume            | `["ReadWriteOnce"]` |
| `persistence.size`          | PVC Size for volume                   | `6Gi`               |
| `persistence.existingClaim` | Specify if the PVC already exists     | `""`                |
| `persistence.annotations`   | Additional custom annotations for PVC | `{}`                |
| `persistence.selector`      | PVC selector                          | `{}`                |


### Database configuration

| Name                      | Description                           | Value            |
| ------------------------- | ------------------------------------- | ---------------- |
| `mysql.enabled`           | Enable MySQL                          | `true`           |
| `mysql.architecture`      | architecture.                         | `standalone`     |
| `mysql.auth.rootPassword` | The password for the MySQL root user. | `""`             |
| `mysql.auth.username`     | The nonroot username of the MySQL.    | `""`             |
| `mysql.auth.password`     | The nonroot password of the MySQL.    | `""`             |
| `mysql.auth.database`     | Create database of the MySQL.         | `""`             |
| `mysql.image.repository`  | mysql repository override             | `kubegems/mysql` |


### External Database configuration

| Name                                         | Description                                                             | Value                 |
| -------------------------------------------- | ----------------------------------------------------------------------- | --------------------- |
| `externalDatabase.enabled`                   | Enable External Database Configuration                                  | `false`               |
| `externalDatabase.host`                      | Database host                                                           | `mysql`               |
| `externalDatabase.port`                      | Database port number                                                    | `3306`                |
| `externalDatabase.username`                  | Non-root username for Concourse                                         | `""`                  |
| `externalDatabase.password`                  | Password for the non-root username for Concourse                        | `""`                  |
| `externalDatabase.database`                  | Concourse database name                                                 | `kubegems`            |
| `externalDatabase.existingSecret`            | Name of an existing secret resource containing the database credentials | `mysql`               |
| `externalDatabase.existingSecretPasswordKey` | Name of an existing secret key containing the database credentials      | `mysql-root-password` |


### RedisCache configuration

| Name                                       | Description                                                                                     | Value                    |
| ------------------------------------------ | ----------------------------------------------------------------------------------------------- | ------------------------ |
| `redis.enabled`                            | Enable redis                                                                                    | `true`                   |
| `redis.architecture`                       | architecture.                                                                                   | `standalone`             |
| `redis.auth.password`                      | The password for the redis,keep emty to use default.                                            | `""`                     |
| `redis.volumePermissions.enabled`          | Enable init container that changes the owner/group of the PV mount point to `runAsUser:fsGroup` | `true`                   |
| `redis.volumePermissions.image.repository` | Repository override                                                                             | `kubegems/bitnami-shell` |
| `redis.image.repository`                   | redis repository override                                                                       | `kubegems/redis`         |


### External RedisCache configuration

| Name                                      | Description                                                          | Value   |
| ----------------------------------------- | -------------------------------------------------------------------- | ------- |
| `externalRedis.enabled`                   | Enable external redis                                                | `false` |
| `externalRedis.host`                      | Redis host                                                           | `redis` |
| `externalRedis.port`                      | Redis port number                                                    | `6379`  |
| `externalRedis.password`                  | Redis password                                                       | `""`    |
| `externalRedis.existingSecret`            | Name of an existing secret resource containing the redis credentials | `""`    |
| `externalRedis.existingSecretPasswordKey` | Name of an existing secret key containing the redis credentials      | `""`    |


### ArgoCD configuration

| Name                                              | Description                                                                | Value                     |
| ------------------------------------------------- | -------------------------------------------------------------------------- | ------------------------- |
| `argo-cd.enabled`                                 | Enable Argo CD                                                             | `true`                    |
| `argo-cd.config.secret.argocdServerAdminPassword` | The password for the ArgoCD server admin user,keep empty to auto-generate. | `""`                      |
| `argo-cd.controller.extraArgs`                    | Extra ArgoCD controller args                                               | `["--redisdb","1"]`       |
| `argo-cd.server.extraArgs`                        | Extra ArgoCD server args                                                   | `["--redisdb","1"]`       |
| `argo-cd.repoServer.extraArgs`                    | Extra ArgoCD repo server args                                              | `["--redisdb","1"]`       |
| `argo-cd.redis.enabled`                           | Disable Argo CD redis to use kubegems redis                                | `false`                   |
| `argo-cd.externalRedis.host`                      | Kubegems Redis host                                                        | `kubegems-redis-headless` |
| `argo-cd.externalRedis.existingSecret`            | Kubegems Redis secret                                                      | `kubegems-redis`          |
| `argo-cd.image.repository`                        | argo-cd repository override                                                | `kubegems/argo-cd`        |
| `argo-cd.redis.image.repository`                  | argocd redis image                                                         | `kubegems/redis`          |


### External ArgoCD configuration

| Name                                       | Description                                                            | Value                                  |
| ------------------------------------------ | ---------------------------------------------------------------------- | -------------------------------------- |
| `externalArgoCD.enabled`                   | Enable external Argo CD                                                | `false`                                |
| `externalArgoCD.address`                   | Argo CD address                                                        | `http://argo-cd-argocd-server.argo-cd` |
| `externalArgoCD.username`                  | Argo CD username                                                       | `admin`                                |
| `externalArgoCD.password`                  | Argo CD password                                                       | `password`                             |
| `externalArgoCD.existingSecret`            | Name of an existing secret resource containing the Argo CD credentials | `""`                                   |
| `externalArgoCD.existingSecretPasswordKey` | Name of an existing secret key containing the Argo CD credentials      | `""`                                   |


### Gitea configuration

| Name                                  | Description                                                       | Value            |
| ------------------------------------- | ----------------------------------------------------------------- | ---------------- |
| `gitea.enabled`                       | Enable Gitea                                                      | `true`           |
| `gitea.memcached.enabled`             | Disable Gitea memcached by default                                | `false`          |
| `gitea.postgresql.enabled`            | Disable Gitea postgresql by default,use built in sqlite3 instead. | `false`          |
| `gitea.gitea.config.database.DB_TYPE` | Use sqlite3 by default                                            | `sqlite3`        |
| `gitea.image.repository`              | gitea repository override                                         | `kubegems/gitea` |


### External Git configuration

| Name                                    | Description                                                        | Value                     |
| --------------------------------------- | ------------------------------------------------------------------ | ------------------------- |
| `externalGit.enabled`                   | Enable external Git                                                | `false`                   |
| `externalGit.address`                   | Git server address                                                 | `https://git.example.com` |
| `externalGit.username`                  | Git username                                                       | `root`                    |
| `externalGit.password`                  | Git password                                                       | `""`                      |
| `externalGit.existingSecret`            | Name of an existing secret resource containing the Git credentials | `""`                      |
| `externalGit.existingSecretPasswordKey` | Name of an existing secret key containing the Git credentials      | `""`                      |


### Chartmuseum configuration

| Name                           | Description                       | Value                  |
| ------------------------------ | --------------------------------- | ---------------------- |
| `chartmuseum.enabled`          | Enable Chartmuseum                | `true`                 |
| `chartmuseum.image.repository` | chartmuseum repository override   | `kubegems/chartmuseum` |
| `chartmuseum.env`              | Chartmuseum environment variables | `1`                    |


### External Chartmuseum configuration

| Name                       | Description                  | Value                             |
| -------------------------- | ---------------------------- | --------------------------------- |
| `externalAppstore.enabled` | Enable external Chartmuseum  | `false`                           |
| `externalAppstore.address` | External Chartmuseum address | `https://chartmuseum.example.com` |


Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
helm install my-release \
  --set kubegemsUsername=admin \
  --set kubegemsPassword=password \
  --set mariadb.auth.rootPassword=secretpassword \
    kubegems/kubegems
```

The above command sets the kubegems administrator account username and password to `admin` and `password` respectively. Additionally, it sets the MariaDB `root` user password to `secretpassword`.

> NOTE: Once this chart is deployed, it is not possible to change the application's access credentials, such as usernames or passwords, using Helm. To change these application credentials after deployment, delete any persistent volumes (PVs) used by the chart and re-deploy it, or use the application's built-in administrative tools if available.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
helm install my-release -f values.yaml kubegems/kubegems
```

> **Tip**: You can use the default [values.yaml](values.yaml)

d

### Additional environment variables

In case you want to add extra environment variables (useful for advanced operations like custom init scripts), you can use the `extraEnvVars` property.

```yaml
kubegems:
  extraEnvVars:
    - name: LOG_LEVEL
      value: error
```

Alternatively, you can use a ConfigMap or a Secret with the environment variables. To do so, use the `extraEnvVarsCM` or the `extraEnvVarsSecret` values.

## License

Copyright &copy; 2022 KubeGems

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
