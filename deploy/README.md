# Deploy

## Charts

The `plugins` dir contains kubegems helm charts.

| name               | description                                                                       |
| ------------------ | --------------------------------------------------------------------------------- |
| `kubegems`         | kubegems core components, installed on the control cluster                        |
| `kubegems-local`   | kubegems edge components, installed on the managed cluster                        |
| `kubegems-install` | kubegems installer operator,to install kubegems components and related components |

## Setup Kubernets Cluster (optional)

### From Kind

Skip this section if you already have a kubernetes cluster.

install kind from: <https://kind.sigs.k8s.io/docs/user/quick-start/#installation>

or

```sh
go install sigs.k8s.io/kind@v0.12.0
```

and create a kind cluster

```sh
sudo kind create cluster --name kubegems --kubeconfig ${HOME}/.kube/config
```

or you can use below config to quick setup a kind cluster.

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
        dataDir: /tmp/etcd # /tmp directory is a tmpfs(in memory),use it for speeding up etcd and lower disk IO.
EOF
sudo kind create cluster --config=kind-tmp.yaml --kubeconfig ${HOME}/.kube/config
```

update cuurent context of kubeconfig to `kind-kubegems`

```sh
kubectl config use-context kind-kubegems
```

### From Kubeadm

[Bootstrapping clusters with kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/)

or using kubeadm with kuberouter for your local environment:

```sh
sudo kubeadm init --pod-network-cidr 10.244.0.0/16
KUBECONFIG=/etc/kubernetes/admin.conf kubectl taint nodes --all node-role.kubernetes.io/master-
KUBECONFIG=/etc/kubernetes/admin.conf kubectl apply -f https://raw.githubusercontent.com/cloudnativelabs/kube-router/master/daemonset/kubeadm-kuberouter.yaml
```

## Deploy KubeGems

Install kubegems installer using helm.

```sh
helm install --namespace kubegems-installer --create-namespace kubegems-installer plugins/kubegems-installer
```

or deploy installer from generated manifests.

```sh
kubectl create namespace kubegems-installer
kubectl apply --namespace kubegems-installer -f https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/installer.yaml
```

Wait until installer is ready.

```sh
kubectl --namespace kubegems-installer get pods
```

Optional: install nginx-ingress-controller and local-path-provisioner if you has no storage plugin or ingress controller installed(from kubeadm):

```sh
kubectl apply -f https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/extends.yaml
```

Deploy kubegems core components:

```sh
kubectl create namespace kubegems
kubectl apply -f https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/kubegems.yaml
```

Note: if you want to customize kubegems version or use a different storageClass,you must download and edit the `kubegems.yaml` file before apply.

```sh
export STORAGE_CLASS=standard   # change to your storageClass
export KUBEGEMS_VERSION=latest  # change to specify kubegems version
curl -sL https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/kubegems.yaml | sed -e "s/local-path/${STORAGE_CLASS}/g" -e "s/main/${KUBEGEMS_VERSION}/g" > kubegems.yaml
kubectl apply -f kubegems.yaml
```

Wait until everything becomes OK.

```sh
kubectl -n kubegems get pods
```

Accessing kubegems dashboard:

```sh
# use nginx-ingress
PORT=$(kubectl -n ingress-nginx get svc nginx-ingress-controller -ojsonpath='{.spec.ports[0].nodePort}')
ADDRESS=$(kubectl -n ingress-nginx get node -ojsonpath='{.items[0].status.addresses[0].address}')
echo http://$ADDRESS:$PORT
```

```sh
# use port-forward
kubectl -n kubegems port-forward svc/kubegems-dashboard 8080:80
```
