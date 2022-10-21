# 本地开发与调试

本文用于参考如何本地调试 kubegems 各个组件，相对于普通应用程序，调试 kubegems 中的组件会有一定的环境依赖和复杂性。

如果要在本地调试 kubegems 集群，你需要在本地构造出所有需要的运行环境。

## 工具准备

1. 已经安装好的 kubegems 集群
2. vscode + vscode golang 插件
3. kubectl
4. [telepresence](https://www.telepresence.io/docs/latest/install/), 一个小工具，能够将本地环境模拟为容器内环境，并且能够重定向容器内流量到本地。

## 安装 kubegems 集群

一般情况下，你仅需要一个 kubernetes 集群，并且按照 [部署](../deploy/README-zh.md) 流程安装好 kubegems。

确保您的 kubectl 可以直接访问到 kubegems 所在的 kubernetes 集群。

kubegems 有这些组件：

| 组件                      | 用途                                                                                                                      |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| kubegems-api              | kubegems 后端的核心服务，提供各种 api.其依赖于 argocd、gitea、mysql、redis、chartmuseum 服务 。                           |
| kubegems-dashboard        | kubegems 的前端服务，其依赖 kubegems-api,kubegems-msgbus ，提供 web UI。                                                  |
| kubegems-msgbus           | kubegems 消息服务，kubegems 中的实时消息通过该服务提供。包含告警信息推送，资源变动推送等。                                |
| kubegems-worker           | kubegems 异步任务服务，用于异步处理耗时较长的任务，定时任务等                                                             |
| kubegems-chartmuseum      | kubegems chart 仓库，主要用于 kubegems 应用商店                                                                           |
| kubegems-local-agent      | kubegems 集群代理，部署在各个被管理集群的 kubegems-local 下。为 kubegems api 提供数据和执行修改。                         |
| kubegems-local-controller | kubegems 集群控制器，部署在各个被管理集群 kubegems-local 下。用于自动管理集群相关状态，包含租户资源限制，网络隔离等功能。 |

正常情况下有这些 pod

```sh
$ kubectl -n kubegems get po
NAME                                               READY   STATUS      RESTARTS      AGE
kubegems-api-db6ccf8df-7mq6t                       1/1     Running     2 (80s ago)   4m25s
kubegems-argo-cd-app-controller-5b9fd69496-k4l6h   1/1     Running     0             4m25s
kubegems-argo-cd-repo-server-7b5b55fb45-gpsbz      1/1     Running     0             4m25s
kubegems-argo-cd-server-85dfdf798-qqn5j            1/1     Running     0             4m25s
kubegems-chartmuseum-d6fddbfb7-zk5xb               1/1     Running     0             4m24s
kubegems-charts-init-v1.22.0-beta.2-q2pxz          0/1     Completed   0             4m25s
kubegems-dashboard-694445cbcb-rnqfd                1/1     Running     0             4m25s
kubegems-gitea-0                                   1/1     Running     0             4m24s
kubegems-msgbus-656768db8d-c8w6h                   1/1     Running     3 (69s ago)   4m25s
kubegems-mysql-0                                   1/1     Running     0             4m24s
kubegems-redis-master-0                            1/1     Running     0             4m24s
kubegems-worker-6c46bb58d6-vh658                   1/1     Running     3 (66s ago)   4m25s
```

## 开始调试

以在 ubuntu linux 上使用 vscode 调试 kubegems-api 为例。

clone 源码:

```sh
git clone https://github.com/kubegems/kubegems.git
cd kubegems
```

kubegems-api 曾经叫 service）入口为 `kubegems service`,这是 kubegems 最核心的服务，你可以通过 help 查看其需要的配置。
代码在 [cmd/apps/service.go](../cmd/apps/service.go)

为了在本地启动调试需要先构建好依赖环境，这里主要是是准备 gitea argocd mysql redis 等配置,在容器中这些配置都通过环境变量和启动参数的方式配置完成。

```sh
$ kubectl -n kubegems get po kubegems-api-db6ccf8df-7mq6t -oyaml
apiVersion: v1
kind: Pod
metadata:
...
  name: kubegems-api-db6ccf8df-7mq6t
  namespace: kubegems
...
spec:
...
  containers:
  - args:
    - service
    - --system-listen=:8080
    - --jwt-cert=/certs/jwt/tls.crt
    - --jwt-key=/certs/jwt/tls.key
    env:
    - name: MYSQL_PASSWORD
      valueFrom:
        secretKeyRef:
          key: mysql-root-password
          name: kubegems-mysql
    - name: REDIS_PASSWORD
      valueFrom:
        secretKeyRef:
          key: redis-password
          name: kubegems-redis
    - name: ARGO_PASSWORD
      valueFrom:
        secretKeyRef:
          key: clearPassword
          name: argocd-secret
    - name: KUBEGEMS_DEBUG
      value: "false"
    - name: LOG_LEVEL
    envFrom:
    - secretRef:
        name: kubegems-config
    image: registry.k8s.fatalc.cn/kubegems/kubegems:v1.22.0-beta.2
    imagePullPolicy: IfNotPresent
    name: api
    ports:
    - containerPort: 8080
      name: http
      protocol: TCP
    - containerPort: 9100
      name: metrics
      protocol: TCP
...
    volumeMounts:
    - mountPath: /app/data
      name: data
    - mountPath: /certs/jwt
      name: jwt-certs
      readOnly: true
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: kube-api-access-v4wsl
      readOnly: true
```

为了在本地也配置这些依赖，你有两种方式：

1. 将集群中的依赖服务使用 NodePort 暴露出来，在本地手动配置这些服务的地址。
2. 使用 telepresence 帮你完成这一切。

我们选择 telepresence，这确实非常方便。

```sh
telepresence intercept -n kubegems kubegems-api --port 8081:80 --env-file=bin/api.env
```

先解释这个命令，这个表示劫持(或者拦截) `-n kubegems` kubegems 命名空间下 `kubegems-api` 的服务(service)的 80 端口并与本地 8081 端口映射，并将容器内的环境变量存储到 `bin/api.env`文件中。

> telepresence 会根据 service 找到对应的 pod，并修改 pod 注入 sidecar。

```sh
$ telepresence intercept -n kubegems kubegems-api --port 8081:80 --env-file=bin/api.env
Launching Telepresence Root Daemon
Need root privileges to run: /usr/local/bin/telepresence daemon-foreground /home/user/.cache/telepresence/logs /home/user/.config/telepresence
Launching Telepresence User Daemon
connector.Connect: traffic manager not found, if it is not installed, please run 'telepresence helm install'
$ telepresence helm install
Telepresence Network is already disconnected
Telepresence Traffic Manager is already disconnected

Traffic Manager installed successfully
# 首次运行会提示在集群内安装 traffic manager，安装完成后再次运行即可。
$ telepresence intercept -n kubegems kubegems-api --port 8081:80 --env-file=bin/api.env
Connected to context kubernetes-admin@kubernetes (https://10.12.32.21:6443)
An update of telepresence from version 2.7.3 to 2.7.6 is available. Please visit https://www.getambassador.io/docs/telepresence/latest/install/upgrade/ for more info.
telepresence: error: rpc error: code = DeadlineExceeded desc = request timed out while waiting for agent kubegems-api.kubegems to arrive

See logs for details (3 errors found): "/home/user/.cache/telepresence/logs/daemon.log"

See logs for details (2 errors found): "/home/user/.cache/telepresence/logs/connector.log"
If you think you have encountered a bug, please run `telepresence gather-logs` and attach the telepresence_logs.zip to your github issue or create a new one: https://github.com/telepresenceio/telepresence/issues/new?template=Bug_report.md .
# 第一次运行会拉取镜像，因耗时较长，会提示上述错误。不要紧张，稍等一下就好。
# 你会看到原来的 kubegems-api pod 从 1/1 变为了 2/2，telepresence 为pod注入了sidecar，用于劫持pod流量到本地。
$ kubectl -n kubegems get po
NAME                                               READY   STATUS            RESTARTS      AGE
kubegems-api-67fb584bb9-rplsb                      0/2     PodInitializing   0             72s
# 等到 Running 后再执行一次
$ kubectl -n kubegems get po
NAME                                               READY   STATUS      RESTARTS      AGE
kubegems-api-67fb584bb9-rplsb                      2/2     Running     0             2m36s
# 看到如下提示就是成功了
$ telepresence intercept -n kubegems kubegems-api --port 8081:80 --env-file=bin/api.env
Using Deployment kubegems-api
intercepted
    Intercept name         : kubegems-api-kubegems
    State                  : ACTIVE
    Workload kind          : Deployment
    Destination            : 127.0.0.1:8081
    Service Port Identifier: http
    Volume Mount Error     : sshfs is not installed on your local machine
    Intercepting           : all TCP requests
Intercepting all traffic to your service. To route a subset of the traffic instead, use a personal intercept. You can enable personal intercepts by authenticating to Ambassador Cloud with "telepresence login".
```

此时流向 kubegems-api pod 的流量被重定向到了本地 8081 端口，这个时候访问 kubegems web 页面会提示错误了，因为本地没有运行 kubegems-api 服务。
并且此时，你的主机能够直接解析和访问到集群的 Pod IP 和 Service IP 了,是不是非常神奇（没错，telepresence 干的）.

```sh
$ nslookup kubegems-gitea-http.kubegems
Server:         127.0.0.53
Address:        127.0.0.53#53

Name:   kubegems-gitea-http.kubegems
Address: 10.244.0.179
$ curl kubegems-gitea-http.kubegems:3000
<!DOCTYPE html>
<html lang="en-US" class="theme-">
...
```

接下来开始配置 vscode debug 来以调试方式启动 kubegems-api

新建文件 `.vscode/launch.json`

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "api",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd",
      "cwd": "${workspaceFolder}",
      "envFile": "${workspaceFolder}/bin/api.env", // telepresence 生成的 env file
      "args": ["service", "--system-listen=:8081"]
    }
  ]
}
```

现在还差 certs ， certs 里面的证书用于生成签名的 token，需要从集群中下载到本地。

> 可以使用 [telepersence volume mounts](https://www.telepresence.io/docs/latest/reference/volume/) 来帮我们挂载证书，但有点曲折，不采用这种方式。

```sh
mkdir -p certs/jwt
kubectl -n kubegems get secrets kubegems-api-jwt -o jsonpath="{.data['tls\.crt']}" | base64 -d >certs/jwt/tls.crt
kubectl -n kubegems get secrets kubegems-api-jwt -o jsonpath="{.data['tls\.key']}" | base64 -d >certs/jwt/tls.key
```

现在已经万事俱备，可以使用 vscode 开始 debug 了。

> 一些时候，你可能需要在 debug 之前使用 `make generate` 来生成一下必要数据

## 结束调试

在 debug 完成后，使用下面命令还原环境：

```sh
telepresence leave kubegems-api-kubegems
```

如果想要移除附加在 pod 上的 sidecar 可以执行：

```sh
telepresence uninstall -a
```
