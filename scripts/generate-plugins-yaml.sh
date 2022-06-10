#! /bin/sh

if [ $# -lt 1 ]; then
  echo "Generate plugins.yaml from plugins"
  echo ""
  echo "Example: $0 --output plugins.yaml plugins/"
  echo ""
  echo "Usage: $0 [options...] <plugins_dir>"
  echo ""
  echo "Options:"
  echo "  --output <output_file> Output file"
  exit 1
fi

cat <<EOF
# This file is generated by generate-plugins-yaml.sh. Do not edit.
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubegems-global-values # don't change this name
data:
  global.imageRegistry: "docker.io"
  global.imageRepository: "kubegems"
  global.storageClass: "local-path"
  global.clusterName: "local-cluster"
  global.kubegemsVersion: "${VERSION:-main}"
EOF

PLUGINS_DIR=$1
for plugin in $(find ${PLUGINS_DIR} -maxdepth 1 -mindepth 1 -type d -not -name common -not -name kubegems -printf '%f\n'); do
  echo "---"
  yq \
    '.apiVersion="plugins.kubegems.io/v1beta1" |
    .kind="Plugin" |
    .metadata.name=.name |
    .metadata.annotations=.annotations |
    .metadata.annotations."plugins.kubegems.io/description"=.description |
    .metadata.annotations."plugins.kubegems.io/appVersion"=.appVersion |
    .spec.disabled=.annotations."plugins.kubegems.io/enabled" != "true" |
    .spec.kind=.annotations."plugins.kubegems.io/kind" // "helm" |
    .spec.installNamespace=.annotations."plugins.kubegems.io/install-namespace" |
    .spec.valuesFrom=[{"kind":"ConfigMap","name":"kubegems-global-values"}] |
    .=pick(["apiVersion","kind","metadata","spec"])
    ' ${PLUGINS_DIR}/$plugin/Chart.yaml
done