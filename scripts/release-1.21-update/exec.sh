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
kubectl get alertmanagerconfigs.monitoring.coreos.com -A -l alertmanagerConfig=gemcloud -oyaml > $dir/amconfigs.yaml
kubectl get prometheusrules.monitoring.coreos.com -A -l prometheusRule=gemcloud -oyaml > $dir/promrules.yaml
kubectl get flows -A -oyaml > $dir/flows.yaml

# update monitor, log rule
go run scripts/release-1.21-update/main.go

# delete old 
kubectl delete alertmanagerconfigs.monitoring.coreos.com -n gemcloud-monitoring-system gemcloud
kubectl delete prometheusrules.monitoring.coreos.com -n gemcloud-monitoring-system gemcloud
kubectl delete prometheusrules.monitoring.coreos.com -n gemcloud-monitoring-system prometheus-recording-rules
