# Use for v1.21.x update to v1.22.x

## v1.22.0-beta.2 to v1.22.0
1. upgrate prometheus operator CRDs
```bash
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.60.1/example/prometheus-operator-crd/monitoring.coreos.com_alertmanagerconfigs.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.60.1/example/prometheus-operator-crd/monitoring.coreos.com_alertmanagers.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.60.1/example/prometheus-operator-crd/monitoring.coreos.com_podmonitors.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.60.1/example/prometheus-operator-crd/monitoring.coreos.com_probes.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.60.1/example/prometheus-operator-crd/monitoring.coreos.com_prometheuses.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.60.1/example/prometheus-operator-crd/monitoring.coreos.com_prometheusrules.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.60.1/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml
kubectl apply --server-side --force-conflicts -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.60.1/example/prometheus-operator-crd/monitoring.coreos.com_thanosrulers.yaml
```
2. all cluster getChannels
3. config mysql connection and saveChannels
4. all cluster updateReceivers
5. check configmap email template. if not exist
```
kubectl apply --server-side  -f deploy/plugins/monitoring/templates/kubegems-email-template.yaml
```
6. restart alertmanager
```
kubectl delete pod -n kubegems-monitoring -l app.kubernetes.io/name=alertmanager
```

7. update monitor_dashboards and monitor_dashboard_tpls of consul
```sql
use kubegems;
SET SQL_SAFE_UPDATES=0;
update monitor_dashboards set graphs = json_replace(graphs, "$[0].promqlGenerator.resource", "consul") where template = "Consul";
update monitor_dashboards set graphs = json_replace(graphs, "$[1].promqlGenerator.resource", "consul") where template = "Consul";
update monitor_dashboards set graphs = json_replace(graphs, "$[2].promqlGenerator.resource", "consul") where template = "Consul";
update monitor_dashboards set graphs = json_replace(graphs, "$[3].promqlGenerator.resource", "consul") where template = "Consul";
update monitor_dashboards set graphs = json_replace(graphs, "$[4].promqlGenerator.resource", "consul") where template = "Consul";

update monitor_dashboard_tpls set graphs = json_replace(graphs, "$[0].promqlGenerator.resource", "consul") where name = "Consul";
update monitor_dashboard_tpls set graphs = json_replace(graphs, "$[1].promqlGenerator.resource", "consul") where name = "Consul";
update monitor_dashboard_tpls set graphs = json_replace(graphs, "$[2].promqlGenerator.resource", "consul") where name = "Consul";
update monitor_dashboard_tpls set graphs = json_replace(graphs, "$[3].promqlGenerator.resource", "consul") where name = "Consul";
update monitor_dashboard_tpls set graphs = json_replace(graphs, "$[4].promqlGenerator.resource", "consul") where name = "Consul";
```

8. dashboard tpl `Container Memory Use Percent`