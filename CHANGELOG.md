## 1.23.6 / 2023-3-2
### Bugfixes 🐞
🐞 fix(cluster): cert expire parse
🐞 fix(alert): show rule group in alert detail
🐞 fix(channel): sync alert rule status

## 1.23.5 / 2023-2-24
### Enhancements 🎈
* 🎈 perf(alert): remove alertlevel value if from promql or logql by @jojotong in https://github.com/kubegems/kubegems/pull/456
### Bugfixes 🐞
* 🐞 fix(alertrule): email secret label shouldn't use globa variable map. by @jojotong in https://github.com/kubegems/kubegems/pull/453

## 1.23.4 / 2023-2-13
### Changes 🛠
* bugfix npe & upgrade configer by @pepesi in https://github.com/kubegems/kubegems/pull/444
### Bugfixes 🐞
* bugfix audit npe when delete tenant member by @pepesi in https://github.com/kubegems/kubegems/pull/446
* 🐞 fix(appmonitor): otel overview ignore nan by @jojotong in https://github.com/kubegems/kubegems/pull/449

## 1.23.3 / 2023-2-9

## 1.23.2 / 2023-2-7
### Enhancements 🎈
* 🎈 perf(otel): use baggage by @jojotong in https://github.com/kubegems/kubegems/pull/435
### Bugfixes 🐞
* 🐞 fix(channel): find alertrule must add 'where', sync map by @jojotong in https://github.com/kubegems/kubegems/pull/433

## 1.23.1 / 2023-2-1
### Enhancements 🎈
* 🎈 perf(script): check alertrule name on 1.23 update by @jojotong in https://github.com/kubegems/kubegems/pull/418
* 🎈 perf(trace): sort by start time desc by @jojotong in https://github.com/kubegems/kubegems/pull/420

## 1.23.0 / 2023-1-30
Kubegems v1.23 is released now. we refactor alert rule and support edge cluster in this version.
If you want to upgrate from v1.22 to v1.23, please refer instruction <https://github.com/kubegems/kubegems/blob/release-1.23/scripts/release-1.23-update/README.md> to migrate.
### Changes 🛠
* Changes related to internationalization, enhancement for site announcement by @pepesi in https://github.com/kubegems/kubegems/pull/239
* refactor(alertrule): store in db and sync to k8s by @jojotong in https://github.com/kubegems/kubegems/pull/371
* disable zap logger sampling by @jojotong in https://github.com/kubegems/kubegems/pull/415
### Features 🎉
* feat(plugin): add plugins configurable and remote repo support by @cnfatal in https://github.com/kubegems/kubegems/pull/241
* Feat aliyun channels by @jojotong in https://github.com/kubegems/kubegems/pull/309
* Feat dashboard muti query targets by @jojotong in https://github.com/kubegems/kubegems/pull/323
* ✨ feat(otel): add otel service appmonitor apis by @jojotong in https://github.com/kubegems/kubegems/pull/331
* ✨ feat(env): search environment by @jojotong in https://github.com/kubegems/kubegems/pull/332
* Feature: Edge Supports by @cnfatal in https://github.com/kubegems/kubegems/pull/333
* ✨ feat(channel): dingding by @jojotong in https://github.com/kubegems/kubegems/pull/344
* add sql and agent client request to opentelemetry by @jojotong in https://github.com/kubegems/kubegems/pull/374
* ✨ feat(worker): add task to sync alert rule state and check config by @jojotong in https://github.com/kubegems/kubegems/pull/379
* ✨ feat(worker): sync system alert rules by @jojotong in https://github.com/kubegems/kubegems/pull/383
### Enhancements 🎈
* kubectl container enhancement;support completion;suppport emoji character; use init job on databse initialize. by @cnfatal in https://github.com/kubegems/kubegems/pull/281
* add otel runtime metric by @jojotong in https://github.com/kubegems/kubegems/pull/336
* 🎈 perf(worker): check alertrule query result by @jojotong in https://github.com/kubegems/kubegems/pull/390
* sync(tpl): add script to export monitor tpls by @jojotong in https://github.com/kubegems/kubegems/pull/396
* 🎈 perf(otel): add option to enable or disable otel by @jojotong in https://github.com/kubegems/kubegems/pull/398
### Bugfixes 🐞
* Bugfix user related by @pepesi in https://github.com/kubegems/kubegems/pull/301
* bugfix, websocket caused memory grow high by @pepesi in https://github.com/kubegems/kubegems/pull/397
* BUGFIX for modify user project role failed by @pepesi in https://github.com/kubegems/kubegems/pull/406
### Others
* docs: add contributors on readme by @LinkMaq in https://github.com/kubegems/kubegems/pull/269

## 1.22.2 / 2022-12-1
### Features 🎉
* ✨ feat(channel): add sendResolved option by @jojotong in https://github.com/kubegems/kubegems/pull/324
### Enhancements 🎈
* import opentelemetry by @jojotong in https://github.com/kubegems/kubegems/pull/325
### Bugfixes 🐞
* 🐞 fix(auth): user auth error print log by @jojotong in https://github.com/kubegems/kubegems/pull/326

## 1.22.1 / 2022-11-17
### Enhancements 🎈
* 🎈 perf(alert): receiver channel set status by @jojotong in https://github.com/kubegems/kubegems/pull/314
* perf(otel): span metrics do not use recording rule by @jojotong in https://github.com/kubegems/kubegems/pull/315
### Bugfixes 🐞
* 🐞 fix(receiver): do not re-gen receiver when delete by @jojotong in https://github.com/kubegems/kubegems/pull/319
### Others
* Set the standard label for spanmetrics by @LinkMaq in https://github.com/kubegems/kubegems/pull/316
* fix nacos template error '-' by @LinkMaq in https://github.com/kubegems/kubegems/pull/317
* Adding missing variables by @LinkMaq in https://github.com/kubegems/kubegems/pull/318
* fix jaeger helm index error by @LinkMaq in https://github.com/kubegems/kubegems/pull/320

## 1.22.0 / 2022-11-09

- Since KubeGems 1.22.0, we had supported the Model Store. User of KubeGems can be download  tens of thousands AI models from HuggingFace and OpenMMLab. And they could be successfully run in Kubernetes easily. 
 
- We have released a new project [ModelX](https://github.com/kubegems/modelx), which is a  repository for AI models.  Model X is based on the design of OCI and Helm Charts. `Modelx Client` makes it easier for developers to package and publish models locally, and `Modelx Server` can be combined with the KubeGems ModelStore to provide more convenient algorithm deployment online service.

    - Download ModelX (https://github.com/kubegems/modelx/releases)

- KubeGems UI support i18n now,   🇨🇳 Chinese(Simplified)、🇭🇰 Chinese(Traditional)、🇺🇸 English 、🇯🇵 Japanese
More languages support are being translated.
- We use our `kubegems/ingress-nginx-operator` to replace old `kubegems/ingress-nginx-operator` to implement tenantgateway.
- We refactor kubegems observability, like alert channel, feishu alerts(more at `kubegems/alertproxy`) and so on.

### Changes 🛠
* remove componentstatus api by @jojotong in https://github.com/kubegems/kubegems/pull/177
* refactor promql tpl to support 3-level directory by @jojotong in https://github.com/kubegems/kubegems/pull/209
* Merge log monitor receiver by @jojotong in https://github.com/kubegems/kubegems/pull/265
* 🦄 refactor(alert): add alert channel in db, remove receiver by @jojotong in https://github.com/kubegems/kubegems/pull/297
### Features 🎉
* feat(gateway): gateway plugin use kubegems/ingress-nginx-operator to replace nginxinc/nginx-ingress-operator by @jojotong in https://github.com/kubegems/kubegems/pull/167
* ✨ feat(gpu): add nvidia dcgm-exporter plugin by @jojotong in https://github.com/kubegems/kubegems/pull/173
* model serving by @cnfatal in https://github.com/kubegems/kubegems/pull/174
* ✨ feat(monitor): add monitor collector status api by @jojotong in https://github.com/kubegems/kubegems/pull/206
* feat: add oauth token and validate api by @jojotong in https://github.com/kubegems/kubegems/pull/208
* feat(dashboard): manage dashboard templates, and import these when init mysql by @jojotong in https://github.com/kubegems/kubegems/pull/210
* ✨ feat(token): user token manage by @jojotong in https://github.com/kubegems/kubegems/pull/212
* ✨ feat(announcement): add announcement api by @jojotong in https://github.com/kubegems/kubegems/pull/215
* ✨ feat(dashboard): add variables by @jojotong in https://github.com/kubegems/kubegems/pull/217
* feat(spm): upgrade jaeger and opentelmetry by @LinkMaq in https://github.com/kubegems/kubegems/pull/219
* ✨ feat(plugin): logging support aws loki storage by @jojotong in https://github.com/kubegems/kubegems/pull/225
* Perfomance kubegems model  by @cnfatal in https://github.com/kubegems/kubegems/pull/235
* feature: support download/upload file from container by @pepesi in https://github.com/kubegems/kubegems/pull/253
* ✨ feat(receiver): add alert proxy receiver for feishu by @jojotong in https://github.com/kubegems/kubegems/pull/282
* ✨ feat(monitor): monitor plugin add alertproxy component by @jojotong in https://github.com/kubegems/kubegems/pull/284
* feat(models): support for modelDeployment recreate by @cnfatal in https://github.com/kubegems/kubegems/pull/298
* Feat channel test api by @jojotong in https://github.com/kubegems/kubegems/pull/299
* ✨ feat(alert): use kubegems email alert template by @jojotong in https://github.com/kubegems/kubegems/pull/230
* feature: support logquery history add time_range, user can reuse time by @pepesi in https://github.com/kubegems/kubegems/pull/307
### Enhancements 🎈
* 🎈 perf(ingress): add ingressClass in plugin ingresses by @jojotong in https://github.com/kubegems/kubegems/pull/166
* add gpu recording rule by @jojotong in https://github.com/kubegems/kubegems/pull/183
* Promql insert labels by @jojotong in https://github.com/kubegems/kubegems/pull/198
* 🎈 perf(metrics): add sumby when query from template by @jojotong in https://github.com/kubegems/kubegems/pull/199
* 🎈 perf(cluster): return client cert expire time by @jojotong in https://github.com/kubegems/kubegems/pull/211
* 🎈 perf(cluster): sync cluster version in worker by @jojotong in https://github.com/kubegems/kubegems/pull/233
* 🔧 build(generate): update release version in deploy yaml and docs by @jojotong in https://github.com/kubegems/kubegems/pull/242
* 🎈 perf(monitor): container tpl use workload variable, rm uniqindex by @jojotong in https://github.com/kubegems/kubegems/pull/249
* 🎈 perf(alert): do not check when delete promql tpl by @jojotong in https://github.com/kubegems/kubegems/pull/251
* change database default collation. for support emoji by @pepesi in https://github.com/kubegems/kubegems/pull/267
* enhancement: get environment resourcequota in concurrcy by @pepesi in https://github.com/kubegems/kubegems/pull/279
* perf(model): add model annotations by @cnfatal in https://github.com/kubegems/kubegems/pull/288
* enhancement: force validate  clustername ^[a-z][-a-z0-9]{0,32}$ by @pepesi in https://github.com/kubegems/kubegems/pull/289
* 🎈 perf(monitot): upgrade alertproxy from v0.1.0 to v0.2.0 by @jojotong in https://github.com/kubegems/kubegems/pull/304
* 🦄 refactor(alert):alert overview use created_at rather than starts_at by @jojotong in https://github.com/kubegems/kubegems/pull/305
* upgrade alertproxy to v0.3.0 by @jojotong in https://github.com/kubegems/kubegems/pull/312
### Bugfixes 🐞
* bugfix(id: 178): add thirdparty crd roles tracked by events by @LinkMaq in https://github.com/kubegems/kubegems/pull/179
* bugfix: imagePullSecrets error by @pepesi in https://github.com/kubegems/kubegems/pull/182
* 🐞 fix(jwt): fix empty jwt payload by @jojotong in https://github.com/kubegems/kubegems/pull/190
* 🐞 fix(prometheus): query should not unescape again by @jojotong in https://github.com/kubegems/kubegems/pull/197
* 🐞 fix(metrics): label query use full vectorselector expr by @jojotong in https://github.com/kubegems/kubegems/pull/205
* 🐞 fix(eventer): scale kube client qps by @jojotong in https://github.com/kubegems/kubegems/pull/248
* Bufix concurrent map by @pepesi in https://github.com/kubegems/kubegems/pull/254
* fix(plugins): add kubegems plugin by mistake by @cnfatal in https://github.com/kubegems/kubegems/pull/260
* fix(installer): too much helm history by @cnfatal in https://github.com/kubegems/kubegems/pull/263
* fix(models): cherry-pick from main by @cnfatal in https://github.com/kubegems/kubegems/pull/268
* fix(image): parse harbor repo with sub project by @cnfatal in https://github.com/kubegems/kubegems/pull/271
* fix: jaeger plugin doesn't upgrade to 1.36.0 by @LinkMaq in https://github.com/kubegems/kubegems/pull/273
* ci flow performance & bugfix cherry-pick from main by @cnfatal in https://github.com/kubegems/kubegems/pull/283
* fix models controller by @cnfatal in https://github.com/kubegems/kubegems/pull/287
* Bugfix userrelated by @pepesi in https://github.com/kubegems/kubegems/pull/303
* 🐞 fix(dashborad): tpl contaienr memory error by @jojotong in https://github.com/kubegems/kubegems/pull/308
* 🐞 fix(logql): regex content use " ` " by @jojotong in https://github.com/kubegems/kubegems/pull/311
### Others
* Add licence script to add copyright in the begin of code by @jojotong in https://github.com/kubegems/kubegems/pull/160
* clean no use code by @pepesi in https://github.com/kubegems/kubegems/pull/180
* dcgm-exporter scrape interval to 15s by @jojotong in https://github.com/kubegems/kubegems/pull/191
* fix(deploy): fix typo in kubegems-mirror.yaml by @itxx00 in https://github.com/kubegems/kubegems/pull/187
* feature: support i18n by @pepesi in https://github.com/kubegems/kubegems/pull/
* New Crowdin updates by @pepesi in https://github.com/kubegems/kubegems/pull/194
* enhancement, use context.Context to determin use which language by @pepesi in https://github.com/kubegems/kubegems/pull/195
* bugfix, i18n can't recognize language correctly by @pepesi in https://github.com/kubegems/kubegems/pull/196
* Update ReadMe by @LinkMaq in https://github.com/kubegems/kubegems/pull/200
* models fix by @cnfatal in https://github.com/kubegems/kubegems/pull/201
* merge release-1.21 back to main by @jojotong in https://github.com/kubegems/kubegems/pull/229

## 1.21.4 / 2022-09-08
### Enhancements 🎈
* fix agent update,add api-resources cache by @cnfatal in https://github.com/kubegems/kubegems/pull/223
* add api-resources filter by @cnfatal in https://github.com/kubegems/kubegems/pull/224
### Bugfixes 🐞
* fix(#222): nacos install default namespace by @LinkMaq in https://github.com/kubegems/kubegems/pull/228

## 1.21.3 / 2022-08-25
### Bugfixes 🐞
* Bugfix image pull secerts error by @pepesi in https://github.com/kubegems/kubegems/pull/213
* 🐞 fix(eventer): drop 'lease' event by @jojotong in https://github.com/kubegems/kubegems/pull/214

## 1.21.2 / 2022-08-09
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

