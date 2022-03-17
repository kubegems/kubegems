# Quick Start

## Deploy

Follow [deploy/README.md](../deploy/README.md) to setup a kubegems cluster.

## Develop

Start develop kubgems components base on above cluster.

Expose depend svc using NodePort in order we can access them locally.

```sh
kubectl --context kind-kubegems -n kubegems expose service kubegems-gitea-http --name=kubegems-gitea --type=NodePort
kubectl --context kind-kubegems -n kubegems expose service kubegems-redis-master --name=kubegems-redis --type=NodePort
kubectl --context kind-kubegems -n kubegems expose service mysql --name=kubegems-mysql --type=NodePort
kubectl --context kind-kubegems -n kubegems expose service kubegems-argocd-server --name=kubegems-argocd --type=NodePort --port=80 --target-port=server
kubectl --context kind-kubegems -n kubegems patch svc kubegems-msgbus --patch='{"spec":{"type":"NodePort"}}'

argocdPort=$(kubectl --context kind-kubegems -n kubegems get svc kubegems-argocd -o jsonpath='{.spec.ports[0].nodePort}')
argocdPassword=$(kubectl --context kind-kubegems -n kubegems get secrets argocd-initial-admin-secret -ogo-template='{{ .data.password | base64decode }}')
redisPort=$(kubectl --context kind-kubegems -n kubegems get svc kubegems-redis -o jsonpath='{.spec.ports[0].nodePort}')
redisPassword=$(kubectl -n kubegems get secrets kubegems-redis -ogo-template='{{ index .data "redis-password" | base64decode }}')
giteaPort=$(kubectl --context kind-kubegems -n kubegems get svc kubegems-gitea -o jsonpath='{.spec.ports[0].nodePort}')
giteaPassword=$(kubectl --context kind-kubegems -n kubegems get secrets kubegems-config -ogo-template='{{ .data.GIT_PASSWORD | base64decode }}')
mysqlPort=$(kubectl --context kind-kubegems -n kubegems get svc kubegems-mysql -o jsonpath='{.spec.ports[0].nodePort}')
mysqlPassword=$(kubectl --context kind-kubegems -n kubegems get secrets mysql -ogo-template='{{ index .data "mysql-root-password" | base64decode }}')
msgbusPort=$(kubectl --context kind-kubegems -n kubegems get svc kubegems-msgbus -o jsonpath='{.spec.ports[0].nodePort}')
nodeAddress=$(kubectl --context kind-kubegems get node -ojsonpath='{.items[0].status.addresses[0].address}')

cat <<EOF | tee config/config.yaml
mysql:
  addr: ${nodeAddress}:${mysqlPort}
  password: ${mysqlPassword}
redis:
  addr: ${nodeAddress}:${redisPort}
  password: ${redisPassword}
git:
  addr: http://${nodeAddress}:${giteaPort}
  password: ${giteaPassword}
argo:
  addr: http://${nodeAddress}:${argocdPort}
  password: ${argocdPassword}
msgbus:
  addr: http://${nodeAddress}:${msgbusPort}
EOF
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
