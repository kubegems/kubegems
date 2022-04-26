#! /bin/sh

usage() {
    echo "Usage: $0 [options] [image]"
    echo "Options:"
    echo "  -h, --help: print this help"
    echo "  -l, --list: list all images"
    echo "  -t, --to: copy all images to target registry. (example: ${DEST_REGISTRY})"
    exit 1
}

DEST_REGISTRY=docker.io/kubegems
ACTION=

# Generate from kubectl on a newly deployed cluster
# kubectl get po --all-namespaces -oyaml | awk 'match($$0,/image:\s"*([a-z0-9:/@.:\-]+)/,i){print i[1]}' | uniq
CUSTOM_IMAGES='
quay.io/argoproj/argo-rollouts:v1.2.0
quay.io/jetstack/cert-manager-controller:v1.8.0
quay.io/jetstack/cert-manager-cainjector:v1.8.0
quay.io/jetstack/cert-manager-webhook:v1.8.0
k8s.gcr.io/ingress-nginx/controller:v1.1.3
docker.io/istio/install-cni:1.11.7
docker.io/istio/operator:1.11.7
docker.io/istio/pilot:1.11.7
docker.io/istio/proxyv2:1.11.7
quay.io/kiali/kiali:v1.38.1
k8s.gcr.io/metrics-server/metrics-server:v0.6.1
quay.io/argoproj/argocd:v2.2.5
ghcr.io/dexidp/dex:v2.30.0
docker.io/library/redis:6.2.6-alpine
ghcr.io/helm/chartmuseum:v0.14.0
docker.io/gitea/gitea:1.16.6
docker.io/bitnami/postgresql:11.11.0-debian-10-r62
docker.io/bitnami/redis:6.2.6-debian-10-r192
docker.io/bitnami/mysql:8.0.28-debian-10-r63
docker.io/rancher/local-path-provisioner:v0.0.22
ghcr.io/banzaicloud/logging-operator:3.17.4
docker.io/grafana/loki:2.4.2
docker.io/grafana/promtail:2.1.0
docker.io/grafana/grafana:8.4.5
docker.elastic.co/elasticsearch/elasticsearch:7.17.1
docker.io/jaegertracing/jaeger-collector:1.30.0
docker.io/jaegertracing/jaeger-es-index-cleaner:1.30.0
docker.io/jaegertracing/jaeger-operator:1.30.0
docker.io/jaegertracing/jaeger-agent:1.30.0
docker.io/jaegertracing/jaeger-query:1.30.0
ghcr.io/jaegertracing/spark-dependencies/spark-dependencies:latest
quay.io/prometheus/alertmanager:v0.24.0
quay.io/prometheus-operator/prometheus-config-reloader:v0.55.0
quay.io/prometheus-operator/prometheus-operator:v0.55.0
quay.io/prometheus/node-exporter:v1.3.1
quay.io/prometheus/prometheus:v2.34.0
quay.io/prometheus-operator/prometheus-config-reloader:v0.55.0
quay.io/kiwigrid/k8s-sidecar:1.15.6
k8s.gcr.io/kube-state-metrics/kube-state-metrics:v2.4.1
'

parsed_images() {
    awk 'match($$0,/image:\s"*([a-z0-9:/@.:\-]+)/,i){print i[1]}' | uniq
}

list_images() {
    for image in ${CUSTOM_IMAGES}; do
        echo ${image}
    done
    # bin/kubegems plugins template deploy/plugins/* | parsed_images
    # bin/kubegems plugins template deploy/plugins-local-stack.yaml | bin/kubegems plugins template - | parsed_images
}

copy_image() {
    tagedimage=${DEST_REGISTRY}/${1##*/}
    echo "copying [${image}] --> [${tagedimage}]"
    if [ "${tagedimage}" = "${image}" ]; then
        echo "skipping [${image}]"
        return
    fi
    skopeo copy docker://${image} docker://${tagedimage}
}

OPTS=$(getopt -o t:,l,h -l to:,list,help -- "$@")
if [ $? != 0 ]; then
    usage
fi
eval set -- "$OPTS"
while true; do
    case $1 in
    -l | --list)
        ACTION=list
        shift
        ;;
    -t | --to)
        DEST_REGISTRY=$2
        ACTION=copy
        shift 2
        ;;
    -h | --help)
        usage
        ;;
    --)
        shift
        break
        ;;
    *)
        echo "unexpected option: $1"
        usage
        ;;
    esac
done

if [ "${ACTION}" = "copy" ]; then
    for image in $(list_images); do
        copy_image ${image}
    done
    exit 0
fi

if [ "${ACTION}" = "list" ]; then
    list_images
    exit 0
fi

usage
