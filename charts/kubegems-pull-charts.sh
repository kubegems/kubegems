#! /bin/sh

set -e
set -x

# Absolute path to this script, e.g. /home/user/bin/foo.sh
SCRIPT=$(readlink -f "$0")
# Absolute path this script is in, thus /home/user/bin
SCRIPTPATH=$(dirname "$SCRIPT")
cd $SCRIPTPATH

# https://gitea.com/gitea/helm-chart/
helm repo add gitea-charts https://dl.gitea.io/charts/
# https://charts.bitnami.com/
helm repo add bitnami https://charts.bitnami.com/bitnami
# https://istio.io/latest/docs/setup/install/helm/
helm repo add istio https://istio-release.storage.googleapis.com/charts
# https://github.com/jaegertracing/helm-charts
helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
# https://github.com/prometheus-community/helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
# https://kiali.io/docs/installation/installation-guide/install-with-helm/
helm repo add kiali https://kiali.org/helm-charts
# https://github.com/NVIDIA/k8s-device-plugin#deployment-via-helm
helm repo add nvdp https://nvidia.github.io/k8s-device-plugin
# https://projectcalico.docs.tigera.io/getting-started/kubernetes/helm
helm repo add projectcalico https://projectcalico.docs.tigera.io/charts
# https://banzaicloud.com/docs/one-eye/logging-operator/install/#helm
helm repo add banzaicloud-stable https://kubernetes-charts.banzaicloud.com
# https://grafana.com/docs/loki/latest/installation/helm/
helm repo add grafana https://grafana.github.io/helm-charts
# https://github.com/kubernetes-sigs/metrics-server#installation
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
# https://github.com/argoproj/argo-helm
helm repo add argo https://argoproj.github.io/argo-helm
# helm repo add jetstack https://charts.jetstack.io
helm repo add jetstack https://charts.jetstack.io

helm dependency build kubegems
helm dependency build kubegems-local

helm pull --untar gitea-charts/gitea
helm pull --untar argo/argo-cd
helm pull --untar argo/argo-rollouts
helm pull --untar jetstack/cert-manager
helm pull --untar bitnami/grafana-operator
helm pull --untar bitnami/nginx-ingress-controller
helm pull --untar jaegertracing/jaeger-operator
# https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-state-metrics
helm pull --untar prometheus-community/kube-state-metrics
# https://github.com/prometheus-operator/prometheus-operator/tree/main/helm
helm pull --untar prometheus-community/kube-prometheus-stack
helm pull --untar prometheus-community/prometheus-node-exporter
helm pull --untar kiali/kiali-server
helm pull --untar nvdp/nvidia-device-plugin
helm pull --untar projectcalico/tigera-operator
helm pull --untar banzaicloud-stable/logging-operator
# https://github.com/grafana/helm-charts/tree/main/charts/loki
helm pull --untar grafana/loki-stack
# https://artifacthub.io/packages/helm/metrics-server/metrics-server
helm pull --untar metrics-server/metrics-server

# local-path-provisioner
# https://github.com/rancher/local-path-provisioner/tree/master/deploy/chart
git clone --depth=1 https://github.com/rancher/local-path-provisioner.git _local-path-provisioner
mv _local-path-provisioner/deploy/chart local-path-provisioner
rm -rf _local-path-provisioner

# istio operator
git clone --depth=1 --branch=release-1.11  https://github.com/istio/istio.git _istio
mv _istio/manifests/charts/istio-operator istio-operator
rm -rf _istio

# https://github.com/tkestack/gpu-manager
# TODO

echo "all charts downloaded"
ls -l -h