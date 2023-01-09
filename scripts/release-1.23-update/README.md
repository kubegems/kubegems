# Upgrade from 1.22 to 1.23

## Monitor plugin
1. update crds
```bash
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.61.1/example/prometheus-operator-crd/monitoring.coreos.com_alertmanagerconfigs.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.61.1/example/prometheus-operator-crd/monitoring.coreos.com_alertmanagers.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.61.1/example/prometheus-operator-crd/monitoring.coreos.com_podmonitors.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.61.1/example/prometheus-operator-crd/monitoring.coreos.com_probes.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.61.1/example/prometheus-operator-crd/monitoring.coreos.com_prometheuses.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.61.1/example/prometheus-operator-crd/monitoring.coreos.com_prometheusrules.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.61.1/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.61.1/example/prometheus-operator-crd/monitoring.coreos.com_thanosrulers.yaml
```
2. upgrate monitor plugin to 43.2.1
```
kubectl patch plugins.plugins.kubegems.io -n kubegems-installer monitoring --type merge -p '{"spec":{"version":"43.2.1"}}'
```
3. goto dashboard to change `externalHost`, `externalPort`
4. patch alertmanager to disable force namespace(because https://github.com/prometheus-community/helm-charts/pull/2882 not merged yet)
```bash
kubectl patch alertmanager -n kubegems-monitoring kube-prometheus-stack-alertmanager --type merge -p '{"spec": {"alertmanagerConfigMatcherStrategy": {"type":"None"}}}'
```

## Opentelemetry plugin
1. upgrate opentelemetry plugin to 0.28.1
```
kubectl patch plugins.plugins.kubegems.io -n kubegems-installer opentelemetry --type merge -p '{"spec":{"version":"0.28.1"}}'
```
2. remove prometheusremotewrite exporter
```
kubectl patch plugins.plugins.kubegems.io -n observability opentelemetry-collector --type json -p '[{"op": "remove", "path": "/spec/values/config/exporters/prometheusremotewrite"}]'
```

## Run script

### Run on manager cluster:

```sh
# switch kubeconfig current context to target cluster
# run migrate
go run ./scripts/release-1.23-update --manager --kubegemsVersion v1.23.0[-xxx]
```

or specify the context name in kubeconfig:

```sh
go run ./scripts/release-1.23-update --context <context_name> --manager --kubegemsVersion v1.23.0[-xxx]
```

To avoid migrate alertrule failed, we should do two things in database manually, after exec `exportOldAlertRulesToDB`:
1. change chinese alertrule name to english
2. update alertrule RedisMemoryHigh's expr and alert levels

### Run on per agent cluster(include manager cluster):

```sh
# switch kubeconfig context to target cluster
go run ./scripts/release-1.23-update --agent --kubegemsVersion v1.23.0[-xxx]
# apply 1.23 installer
kubectl apply -f https://github.com/kubegems/kubegems/raw/release-1.23/deploy/installer.yaml
```
