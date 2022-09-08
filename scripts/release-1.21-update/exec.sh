set -x

kubectl scale deployment -n gemcloud-monitoring-system prometheus-operator --replicas 0

# prometheus-operator crds
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.56.2/example/prometheus-operator-crd/monitoring.coreos.com_alertmanagerconfigs.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.56.2/example/prometheus-operator-crd/monitoring.coreos.com_alertmanagers.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.56.2/example/prometheus-operator-crd/monitoring.coreos.com_podmonitors.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.56.2/example/prometheus-operator-crd/monitoring.coreos.com_probes.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.56.2/example/prometheus-operator-crd/monitoring.coreos.com_prometheuses.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.56.2/example/prometheus-operator-crd/monitoring.coreos.com_prometheusrules.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.56.2/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.56.2/example/prometheus-operator-crd/monitoring.coreos.com_thanosrulers.yaml

# backup
dir="backup/$(kubectl config current-context)"
mkdir -p $dir

kubectl delete servicemonitors.monitoring.coreos.com -n gemcloud-monitoring-system --all
kubectl delete podmonitors.monitoring.coreos.com -n cert-manager cert-manager

# prometheus
kubectl get prometheus -n gemcloud-monitoring-system gemcloud -oyaml > $dir/prometheus.yaml
kubectl delete -f $dir/prometheus.yaml

kubectl get alertmanager -n gemcloud-monitoring-system gemcloud -oyaml > $dir/alertmanager.yaml
kubectl delete -f $dir/alertmanager.yaml

kubectl get grafana -n gemcloud-monitoring-system -oyaml > $dir/grafana.yaml
kubectl delete -f $dir/grafana.yaml

kubectl get ds -n gemcloud-monitoring-system node-exporter -oyaml > $dir/node.yaml
kubectl delete -f $dir/node.yaml

kubectl get deployments.apps -n gemcloud-monitoring-system kube-state-metrics -oyaml > $dir/state.yaml
kubectl delete -f $dir/state.yaml

kubectl get deployments.apps -n gemcloud-monitoring-system grafana-operator -oyaml > $dir/grafana-operator.yaml
kubectl delete -f $dir/grafana-operator.yaml

kubectl get deployments.apps -n gemcloud-monitoring-system prometheus-operator -oyaml > $dir/prometheus-operator.yaml
kubectl delete -f $dir/prometheus-operator.yaml


kubectl get alertmanagerconfigs.monitoring.coreos.com -A -l alertmanagerConfig=gemcloud -oyaml > $dir/amconfigs.yaml
kubectl get prometheusrules.monitoring.coreos.com -A -l prometheusRule=gemcloud -oyaml > $dir/promrules.yaml
kubectl get flows -A -oyaml > $dir/flows.yaml

# update monitor, log rule
go run scripts/release-1.21-update/main.go

# delete old 
kubectl delete alertmanagerconfigs.monitoring.coreos.com -n gemcloud-monitoring-system gemcloud
kubectl delete prometheusrules.monitoring.coreos.com -n gemcloud-monitoring-system gemcloud
kubectl delete prometheusrules.monitoring.coreos.com -n gemcloud-monitoring-system prometheus-recording-rules
