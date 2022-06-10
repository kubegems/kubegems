# 部署

## charts

`plugins` 目录包含 kubegems helm charts.

| 名称                 | 描述                                            |
| -------------------- | ----------------------------------------------- |
| `kubegems-installer` | kubegems 安装程序，安装 kubegems 组件及相关组件 |
| `kubegems`           | kubegems 核心组件，包含 dashboard 及其后端      |
| `kubegems-local`     | kubegems 边缘组件，安装在托管集群上             |

## 安装 Kubernetes 集群（可选）

### 使用 Kind 安装

如果您已经拥有 Kubernetes 集群，请跳过此部分。

从以下位置安装 Kind：<https://kind.sigs.k8s.io/docs/user/quick-start/#installation>

或者

```sh
go install sigs.k8s.io/kind@v0.12.0
```

并创建一个 kind 集群

```sh
sudo kind create cluster --name kubegems --kubeconfig ${HOME}/.kube/config
```

或者你可以使用下面的配置来快速设置一个集群。

```sh
cat <<EOF | tee kind-tmp.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: kubegems
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 30000
        hostPort: 30000
kubeadmConfigPatches:
  - |
    apiVersion: kubeadm.k8s.io/v1beta2
    kind: ClusterConfiguration
    etcd:
      local:
        dataDir: /tmp/etcd # /tmp 目录是一个 tmpfs（在内存中），用于加速 etcd 和降低磁盘 IO。但是不能持久化，所以在每次重启时会丢失。
EOF
sudo kind create cluster --config=kind-tmp.yaml --kubeconfig ${HOME}/.kube/config
```

将 kubeconfig 的当前上下文更新为 `kind-kubegems`

```sh
kubectl config use-context kind-kubegems
```

### 使用 Kubeadm 安装

[使用 kubeadm 引导集群](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/)

或将 kubeadm 与 kuberouter 一起用于您的本地环境：

```sh
sudo kubeadm init --pod-network-cidr 10.244.0.0/16
KUBECONFIG=/etc/kubernetes/admin.conf kubectl taint nodes --all node-role.kubernetes.io/master-
KUBECONFIG=/etc/kubernetes/admin.conf kubectl apply -f https://raw.githubusercontent.com/cloudnativelabs/kube-router/master/daemonset/kubeadm-kuberouter.yaml
```

## 部署 KubeGems

安装 kubegems installer:

```sh
kubectl create namespace kubegems-installer
kubectl apply -f https://github.com/kubegems/kubegems/raw/main/deploy/installer.yaml
```

等待安装程序准备就绪。

```sh
kubectl --namespace bundle-controller get pods
```

可选项:

1. 如果没有 ingress controller，可以安装 nginx-ingress-controller:

  ```sh
  kubectl create namespace ingress-nginx
  kubectl apply -f https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/addon-nginx-ingress.yaml
  ```

1. 如果没有CSI插件，可以安装 local-path-provisioner:

  ```sh
  kubectl create namespace local-path-provisioner
  kubectl apply -f https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/addon-local-path-provisioner.yaml
  ```

部署 kubegems 核心组件：

```sh
kubectl create namespace kubegems
kubectl apply -f https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/kubegems.yaml
```

如果您的网络在获取 docker.io quay.io gcr.io 上的镜像时较为缓慢，可以使用我们在阿里云上的镜像：

```sh
kubectl apply -f https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/kubegems-mirror.yaml
```

注意：如果您想自定义 kubegems 版本或使用不同的 storageClass，您必须在 apply 前下载并编辑 `kubegems.yaml` 文件。

```sh
export STORAGE_CLASS=standard   # change to your storageClass
export KUBEGEMS_VERSION=latest  # change to specify kubegems version
curl -sL https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/kubegems.yaml | sed -e "s/local-path/${STORAGE_CLASS}/g" -e "s/main/${KUBEGEMS_VERSION}/g" > kubegems.yaml
kubectl apply -f kubegems.yaml
```

等到一切正常。

```sh
kubectl -n kubegems get pod
```

访问 kubegems 仪表板：

```sh
# 使用 nginx ingress
PORT=$(kubectl -n ingress-nginx get svc nginx-ingress-controller -ojsonpath='{.spec.ports[0].nodePort}')
ADDRESS=$(kubectl -n ingress-nginx get node -ojsonpath='{.items[0].status.addresses[0].address}')
echo http://$ADDRESS:$PORT
```

```sh
# 使用端口转发
kubectl -n kubegems port-forward svc/kubegems-dashboard 8080:80
```
