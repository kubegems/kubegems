# kubegems-installer

%%DESCRIPTION%% (check existing examples)

## TL;DR

```console
$ helm repo add kubegems https://charts.kubegems.io/kubegems
$ helm install my-release kubegems/kubegems-installer
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
helm install my-release kubegems/kubegems-installer
```

The command deploys kubegems-installer on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

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


### installer Parameters

| Name                                              | Description                                                                                         | Value               |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------- | ------------------- |
| `installer.image.registry`                        | installer image registry                                                                            | `docker.io`         |
| `installer.image.repository`                      | installer image repository                                                                          | `kubegems/kubegems` |
| `installer.image.tag`                             | installer image tag (immutable tags are recommended)                                                | `latest`            |
| `installer.image.pullPolicy`                      | installer image pull policy                                                                         | `IfNotPresent`      |
| `installer.image.pullSecrets`                     | installer image pull secrets                                                                        | `[]`                |
| `installer.image.debug`                           | Enable installer image debug mode                                                                   | `false`             |
| `installer.replicaCount`                          | Number of installer replicas to deploy                                                              | `1`                 |
| `installer.containerPorts.probe`                  | installer probe container port                                                                      | `8080`              |
| `installer.livenessProbe.enabled`                 | Enable livenessProbe on installer containers                                                        | `true`              |
| `installer.livenessProbe.initialDelaySeconds`     | Initial delay seconds for livenessProbe                                                             | `5`                 |
| `installer.livenessProbe.periodSeconds`           | Period seconds for livenessProbe                                                                    | `10`                |
| `installer.livenessProbe.timeoutSeconds`          | Timeout seconds for livenessProbe                                                                   | `5`                 |
| `installer.livenessProbe.failureThreshold`        | Failure threshold for livenessProbe                                                                 | `6`                 |
| `installer.livenessProbe.successThreshold`        | Success threshold for livenessProbe                                                                 | `1`                 |
| `installer.readinessProbe.enabled`                | Enable readinessProbe on installer containers                                                       | `true`              |
| `installer.readinessProbe.initialDelaySeconds`    | Initial delay seconds for readinessProbe                                                            | `5`                 |
| `installer.readinessProbe.periodSeconds`          | Period seconds for readinessProbe                                                                   | `10`                |
| `installer.readinessProbe.timeoutSeconds`         | Timeout seconds for readinessProbe                                                                  | `5`                 |
| `installer.readinessProbe.failureThreshold`       | Failure threshold for readinessProbe                                                                | `6`                 |
| `installer.readinessProbe.successThreshold`       | Success threshold for readinessProbe                                                                | `1`                 |
| `installer.startupProbe.enabled`                  | Enable startupProbe on installer containers                                                         | `false`             |
| `installer.startupProbe.initialDelaySeconds`      | Initial delay seconds for startupProbe                                                              | `5`                 |
| `installer.startupProbe.periodSeconds`            | Period seconds for startupProbe                                                                     | `10`                |
| `installer.startupProbe.timeoutSeconds`           | Timeout seconds for startupProbe                                                                    | `5`                 |
| `installer.startupProbe.failureThreshold`         | Failure threshold for startupProbe                                                                  | `6`                 |
| `installer.startupProbe.successThreshold`         | Success threshold for startupProbe                                                                  | `1`                 |
| `installer.customLivenessProbe`                   | Custom livenessProbe that overrides the default one                                                 | `{}`                |
| `installer.customReadinessProbe`                  | Custom readinessProbe that overrides the default one                                                | `{}`                |
| `installer.customStartupProbe`                    | Custom startupProbe that overrides the default one                                                  | `{}`                |
| `installer.resources.limits`                      | The resources limits for the installer containers                                                   | `{}`                |
| `installer.resources.requests`                    | The requested resources for the installer containers                                                | `{}`                |
| `installer.podSecurityContext.enabled`            | Enabled installer pods' Security Context                                                            | `false`             |
| `installer.podSecurityContext.fsGroup`            | Set installer pod's Security Context fsGroup                                                        | `1001`              |
| `installer.containerSecurityContext.enabled`      | Enabled installer containers' Security Context                                                      | `false`             |
| `installer.containerSecurityContext.runAsUser`    | Set installer containers' Security Context runAsUser                                                | `1001`              |
| `installer.containerSecurityContext.runAsNonRoot` | Set installer containers' Security Context runAsNonRoot                                             | `true`              |
| `installer.leaderElection.enabled`                | Enable leader election                                                                              | `true`              |
| `installer.logLevel`                              | Log level                                                                                           | `debug`             |
| `installer.existingConfigmap`                     | The name of an existing ConfigMap with your custom configuration for installer                      | `nil`               |
| `installer.command`                               | Override default container command (useful when using custom images)                                | `[]`                |
| `installer.args`                                  | Override default container args (useful when using custom images)                                   | `[]`                |
| `installer.hostAliases`                           | installer pods host aliases                                                                         | `[]`                |
| `installer.podLabels`                             | Extra labels for installer pods                                                                     | `{}`                |
| `installer.podAnnotations`                        | Annotations for installer pods                                                                      | `{}`                |
| `installer.podAffinityPreset`                     | Pod affinity preset. Ignored if `installer.affinity` is set. Allowed values: `soft` or `hard`       | `""`                |
| `installer.podAntiAffinityPreset`                 | Pod anti-affinity preset. Ignored if `installer.affinity` is set. Allowed values: `soft` or `hard`  | `soft`              |
| `installer.nodeAffinityPreset.type`               | Node affinity preset type. Ignored if `installer.affinity` is set. Allowed values: `soft` or `hard` | `""`                |
| `installer.nodeAffinityPreset.key`                | Node label key to match. Ignored if `installer.affinity` is set                                     | `""`                |
| `installer.nodeAffinityPreset.values`             | Node label values to match. Ignored if `installer.affinity` is set                                  | `[]`                |
| `installer.enableAffinity`                        | If enabled Affinity for installer pods assignment                                                   | `false`             |
| `installer.affinity`                              | Affinity for installer pods assignment                                                              | `{}`                |
| `installer.nodeSelector`                          | Node labels for installer pods assignment                                                           | `{}`                |
| `installer.tolerations`                           | Tolerations for installer pods assignment                                                           | `[]`                |
| `installer.updateStrategy.type`                   | installer statefulset strategy type                                                                 | `RollingUpdate`     |
| `installer.priorityClassName`                     | installer pods' priorityClassName                                                                   | `""`                |
| `installer.schedulerName`                         | Name of the k8s scheduler (other than default) for installer pods                                   | `""`                |
| `installer.lifecycleHooks`                        | for the installer container(s) to automate configuration before or after startup                    | `{}`                |
| `installer.extraEnvVars`                          | Array with extra environment variables to add to installer nodes                                    | `[]`                |
| `installer.extraEnvVarsCM`                        | Name of existing ConfigMap containing extra env vars for installer nodes                            | `nil`               |
| `installer.extraEnvVarsSecret`                    | Name of existing Secret containing extra env vars for installer nodes                               | `nil`               |
| `installer.extraVolumes`                          | Optionally specify extra list of additional volumes for the installer pod(s)                        | `[]`                |
| `installer.extraVolumeMounts`                     | Optionally specify extra list of additional volumeMounts for the installer container(s)             | `[]`                |
| `installer.sidecars`                              | Add additional sidecar containers to the installer pod(s)                                           | `{}`                |
| `installer.initContainers`                        | Add additional init containers to the installer pod(s)                                              | `{}`                |


### Agent Metrics parameters

| Name                                                 | Description                                                                 | Value                    |
| ---------------------------------------------------- | --------------------------------------------------------------------------- | ------------------------ |
| `installer.metrics.enabled`                          | Create a service for accessing the metrics endpoint                         | `true`                   |
| `installer.metrics.service.type`                     | controller metrics service type                                             | `ClusterIP`              |
| `installer.metrics.service.port`                     | controller metrics service HTTP port                                        | `9100`                   |
| `installer.metrics.service.nodePort`                 | Node port for HTTP                                                          | `""`                     |
| `installer.metrics.service.clusterIP`                | controller metrics service Cluster IP                                       | `""`                     |
| `installer.metrics.service.extraPorts`               | Extra ports to expose (normally used with the `sidecar` value)              | `[]`                     |
| `installer.metrics.service.loadBalancerIP`           | controller metrics service Load Balancer IP                                 | `""`                     |
| `installer.metrics.service.loadBalancerSourceRanges` | controller metrics service Load Balancer sources                            | `[]`                     |
| `installer.metrics.service.externalTrafficPolicy`    | controller metrics service external traffic policy                          | `Cluster`                |
| `installer.metrics.service.annotations`              | Additional custom annotations for controller metrics service                | `{}`                     |
| `installer.metrics.serviceMonitor.enabled`           | Specify if a servicemonitor will be deployed for prometheus-operator        | `true`                   |
| `installer.metrics.serviceMonitor.jobLabel`          | Specify the jobLabel to use for the prometheus-operator                     | `app.kubernetes.io/name` |
| `installer.metrics.serviceMonitor.honorLabels`       | Honor metrics labels                                                        | `false`                  |
| `installer.metrics.serviceMonitor.selector`          | Prometheus instance selector labels                                         | `{}`                     |
| `installer.metrics.serviceMonitor.scrapeTimeout`     | Timeout after which the scrape is ended                                     | `""`                     |
| `installer.metrics.serviceMonitor.interval`          | Scrape interval. If not set, the Prometheus default scrape interval is used | `""`                     |
| `installer.metrics.serviceMonitor.additionalLabels`  | Used to pass Labels that are required by the installed Prometheus Operator  | `{}`                     |
| `installer.metrics.serviceMonitor.metricRelabelings` | Specify additional relabeling of metrics                                    | `{}`                     |
| `installer.metrics.serviceMonitor.relabelings`       | Specify general relabeling                                                  | `{}`                     |


### RBAC Parameters

| Name                    | Description                                                   | Value  |
| ----------------------- | ------------------------------------------------------------- | ------ |
| `rbac.create`           | Specifies whether RBAC resources should be created            | `true` |
| `rbac.useClusterAdmin`  | clusterrolbinding to cluster-admin instead create clusterrole | `true` |
| `serviceAccount.create` | Specifies whether a ServiceAccount should be created          | `true` |
| `serviceAccount.name`   | The name of the ServiceAccount to use.                        | `""`   |


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
