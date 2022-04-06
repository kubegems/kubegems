# Quick Start

## Deploy

Follow [deploy/README.md](../deploy/README.md) to setup a kubegems cluster.

## Develop

Start develop kubgems components base on above cluster.

Expose depend svc using NodePort in order we can access them locally.

```sh
sh scripts/generate-config-from-cluster.sh
```

> make sure the `config/config.yaml` located on the project root.

### Using vscode

vscode setting

```json
// .vscode/launch.json
{
  "name": "api",
  "type": "go",
  "request": "launch",
  "mode": "auto",
  "program": "${workspaceFolder}/cmd",
  "cwd": "${workspaceFolder}",
  "args": ["service"],
  "env": {
    "GEMS_PPROF_PORT": ":6060",
    "EXPORTER_LISTEN": ":9100"
  }
}
```

You can use the [Bridge to Kubernetes](https://marketplace.visualstudio.com/items?itemName=mindaro.mindaro) to redirect remote traffic to the local debugable service.

```json
// .vscode/tasks.json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "bridge-to-kubernetes.resource",
      "type": "bridge-to-kubernetes.resource",
      "resource": "kubegems-api",
      "resourceType": "service",
      "ports": [8080],
      "targetCluster": "kind-kubegems",
      "targetNamespace": "kubegms",
      "useKubernetesServiceEnvironmentVariables": true
    }
  ]
}
```

### Manually

```sh
go run cmd/main.go
```
