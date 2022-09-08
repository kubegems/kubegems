## What's Changed
### Enhancements 🎈
* fix agent update,add api-resources cache by @cnfatal in https://github.com/kubegems/kubegems/pull/223
* add api-resources filter by @cnfatal in https://github.com/kubegems/kubegems/pull/224
### Bugfixes 🐞
* fix(#222): nacos install default namespace by @LinkMaq in https://github.com/kubegems/kubegems/pull/228


## 1.21.4 / 2022-08-09
### Bugfixes 🐞
* 🐞 fix(webhook): ingress api version error by @jojotong in https://github.com/kubegems/kubegems/pull/171
* 🐞 fix(cluster): apiserver version should from k8s, not db by @jojotong in https://github.com/kubegems/kubegems/pull/172
* 🐞 fix(apiresource): handle apiresource group failed error by @jojotong in https://github.com/kubegems/kubegems/pull/175

 
## 1.21.1 / 2022-07-22
### Enhancements 🎈
* 🐞 fix(log): alert duration in template limit to 10m by @jojotong in https://github.com/kubegems/kubegems/pull/152
* 🎈 perf(logging): disable tls in logging-operator by @jojotong in https://github.com/kubegems/kubegems/pull/157
### Bugfixes 🐞
* bugfix; error handle default image registry; by @pepesi in https://github.com/kubegems/kubegems/pull/153
* fix(plugin): #155 gpu can't regist device by @LinkMaq in https://github.com/kubegems/kubegems/pull/159
* 🐞 fix(workload): workload list istio-inject sort error by @jojotong in https://github.com/kubegems/kubegems/pull/161

## 1.21.0 / 2022-07-08

This release brings new plugin management and observability features live. Now you can use the plugin CRD to enable and uninstall platform plugins. For observability, we provide a series of new functions such as access center, monitoring dashboard, log alert, etc.

### Features 🎉
* ✨ feat(monitor): add log and event template by @jojotong in #59
* feat log receiver and alert by @jojotong in #65
* ✨ feat(observability): add dashboard and labelname api by @jojotong in #69
* ✨ feat(monitor): support unit in promql query and dashboard by @jojotong in #75
* ✨ feat(plugin): add logging and eventer plugins by @jojotong in #77
* feat(plugin): add 6 plugins in kuebgems-stack by @LinkMaq in #81
* feat(plugins): split all in one plugins by @cnfatal in #91
* feat(nacos): add nacos plugins for application configure management on kubegems by @pepesi in #90
* Feature nacos client by @pepesi in #113
* batch create applications by @cnfatal in #139
### Enhancements 🎈
* 🎈 perf(plugin): finish monitor plugin transfer by @jojotong in #73
* perf(plugin): add appversion by @cnfatal in #99
* 🎈 perf(gateway): specify different image tag by ingressclass version by @jojotong in #102
* 🎈 perf(gateway): update to v0.5.2 to support workload extra labels by @jojotong in #106
* 🎈 perf(logging): store alert rule in new configmap, to avoid overwrit… by @jojotong in #129
* fix(otel): otlp metrics remotewrite to prometheus by @LinkMaq in #141
* 🎈 perf(alert): alert group show raw promql and logging by @jojotong in #142
### Bugfixes 🐞
* 🐞 fix(observe): promql generator bug by @jojotong in #70
* 🐞 fix(plugin): monitor plugin add promrule and amconfig CRD by @jojotong in #74
* fix(plugins): can't read argocd admin password by @LinkMaq in #80
* fix(charts): kubegems argocd "NOAUTH" and "Token Expire" by @cnfatal in #84
* fix(deploy): add cluster by @cnfatal in #94
* fix(plugins): nacos use helm by @cnfatal in #98
* fix(charts):  Organize the plugins catalog by @LinkMaq in #97
* fix(charts): optimize opentelemetry servicemonitor by @LinkMaq in #103
* bugfix: resolve #s/87 by @pepesi in #107
* fix(charts): opentelemetry export to jaeger by @LinkMaq in #108
* fix(charts): otel nil pointer address by @LinkMaq in #110
* fix(plugins): image registry customize by @cnfatal in #112
* fix(charts): openkruise support and charts describe by @LinkMaq in #114
* fix(charts): can't find openkruise on charts repo by @LinkMaq in #115
* bugfix; environment cache_key error by @pepesi in #127
* 🐞 fix(gateway): use nginx-ingress 2.0.0 on k8s 1.22+ by @jojotong in #134
* 🐞 fix(loki): useExistingAlertingGroup to replace build-in alertingroups by @jojotong in #137
* fix(patch): unable to update some fileds in server side apply by @cnfatal in #148
### Others
* observalibity features by @jojotong in #56
* makefile support tags for condition build by @pepesi in #111
* 📃 docs: update readme,, contributing, code conduct docs by @jojotong in #131

