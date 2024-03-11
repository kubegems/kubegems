{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubegems.secret.fullname" -}}
{{- printf "%s-config" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified mysql name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "kubegems.mysql.fullname" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" "mysql" "chartValues" .Values.mysql "context" $) -}}
{{- end -}}

{{/*
Return the proper database host
*/}}
{{- define "kubegems.database.host" -}}
{{- if .Values.mysql.enabled -}}
    {{- printf "%s-headless" (include "kubegems.mysql.fullname" .)  | trunc 63 | trimSuffix "-" -}}
{{- else if .Values.externalDatabase.enabled -}}
    {{- .Values.externalDatabase.host -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper database port
*/}}
{{- define "kubegems.database.port" -}}
{{- if .Values.mysql.enabled -}}
    {{- .Values.mysql.primary.service.port -}}
{{- else if and .Values.externalDatabase.enabled .Values.externalDatabase.port -}}
    {{- .Values.externalDatabase.port -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper database
*/}}
{{- define "kubegems.database.database" -}}
{{- if .Values.mysql.enabled -}}
    {{- if .Values.mysql.auth.database -}}
        {{- .Values.mysql.auth.database -}}
    {{- else -}}
        {{- /*keep empty,use default*/ -}}
    {{- end -}}
{{- else if and .Values.externalDatabase.enabled .Values.externalDatabase.database -}}
    {{- .Values.externalDatabase.database -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper database username
*/}}
{{- define "kubegems.database.username" -}}
{{- if .Values.mysql.enabled -}}
    {{- if .Values.mysql.auth.username -}}
        {{- .Values.mysql.auth.username -}}
    {{- else -}}
        {{- /*keep empty,use default*/ -}}
    {{- end -}}
{{- else if .Values.externalDatabase.enabled -}}
    {{- .Values.externalDatabase.username -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper database password
*/}}
{{- define "kubegems.database.password" -}}
{{- if .Values.mysql.enabled -}}
    {{ .Values.mysql.auth.password }}
{{- else if .Values.externalDatabase.enabled -}}
    {{- .Values.externalDatabase.password -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper database password secret
*/}}
{{- define "kubegems.database.password.secret" -}}
{{- if .Values.mysql.enabled -}}
    {{- if .Values.mysql.auth.existingSecret -}}
        {{- .Values.mysql.auth.existingSecret -}}
    {{- else -}}
        {{- include "kubegems.mysql.fullname" . -}}
    {{- end -}}
{{- else if .Values.externalDatabase.enabled -}}
    {{- .Values.externalDatabase.existingSecret -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper database password secret key
*/}}
{{- define "kubegems.database.password.secret.key" -}}
{{- if .Values.mysql.enabled -}}
    {{- printf "%s" "mysql-root-password" -}}
{{- else if and .Values.externalDatabase.enabled .Values.externalDatabase.existingSecret -}}
    {{- if .Values.externalDatabase.existingSecretPasswordKey -}}
        {{- .Values.externalDatabase.existingSecretPasswordKey -}}
    {{- else -}}
        {{- printf "%s" "database-password" -}}
    {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified redis name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "kubegems.redis.fullname" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" "redis" "chartValues" .Values.redis "context" $) -}}
{{- end -}}

{{/*
Return the proper redis host
*/}}
{{- define "kubegems.redis.host" -}}
{{- if .Values.redis.enabled -}}
    {{- printf "%s-headless" (include "kubegems.redis.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- else if .Values.externalRedis.enabled -}}
    {{- .Values.externalRedis.host -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper redis port
*/}}
{{- define "kubegems.redis.port" -}}
{{- if .Values.redis.enabled -}}
{{- .Values.redis.master.containerPorts.redis -}}
{{- else if and .Values.externalRedis.enabled .Values.externalRedis.port -}}
{{- .Values.externalRedis.port -}}
{{- end -}}
{{- end -}}

{{- define "kubegems.redis.address" -}}
{{- if (include "kubegems.redis.host" .) -}}
    {{- printf "%s:%s" (include "kubegems.redis.host" .) (include "kubegems.redis.port" .) -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper database password
*/}}
{{- define "kubegems.redis.password" -}}
{{- if .Values.redis.enabled -}}
    {{/*use secret ref,not dictly password*/}}
{{- else if .Values.externalRedis.enabled -}}
    {{- .Values.externalRedis.password -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper redis password secret
*/}}
{{- define "kubegems.redis.password.secret" -}}
{{- if .Values.redis.enabled -}}
    {{- include "kubegems.redis.fullname" . -}}
{{- else if .Values.externalRedis.enabled -}}
    {{- .Values.externalRedis.existingSecret -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper redis password secret key
*/}}
{{- define "kubegems.redis.password.secret.key" -}}
{{- if .Values.redis.enabled -}}
    {{- printf "%s" "redis-password" -}}
{{- else if and .Values.externalRedis.enabled .Values.externalRedis.existingSecret -}}
    {{- if .Values.externalRedis.existingSecretPasswordKey -}}
        {{ .Values.externalRedis.existingSecretPasswordKey }}
    {{- else -}}
        {{- printf "%s" "redis-password" -}}
    {{- end -}}
{{- end -}}
{{- end -}}



{{/*
Create a default fully qualified argocd name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "kubegems.argocd.fullname" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" "argo-cd" "chartValues" (index .Values "argo-cd") "context" $) -}}
{{- end -}}

{{/*
Return the proper argocd address
*/}}
{{- define "kubegems.argocd.address" -}}
{{- if (index .Values "argo-cd" "enabled") -}}
    {{- printf "http://%s-server:" (include "kubegems.argocd.fullname" .) -}}{{- (index .Values "argo-cd" "server" "service" "ports" "http") }}
{{- else if .Values.externalArgoCD.enabled -}}
    {{- .Values.externalArgoCD.address -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper argocd username
*/}}
{{- define "kubegems.argocd.username" -}}
{{- if (index .Values "argo-cd" "enabled") -}}
    {{- "admin" -}}
{{- end -}}
{{- if .Values.externalArgoCD.enabled -}}
    {{- .Values.externalArgoCD.username -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper argocd password secret
*/}}
{{- define "kubegems.argocd.password" -}}
{{- with index .Values "argo-cd" -}}
{{- if and .enabled (not .config.secret.create) -}}
    {{- .config.secret.argocdServerAdminPassword -}}
{{- else if and $.Values.externalArgoCD.enabled -}}
    {{- $.Values.externalArgoCD.password -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper argocd password secret
*/}}
{{- define "kubegems.argocd.password.secret" -}}
{{- $argocd := index .Values "argo-cd" -}}
{{- if and $argocd.enabled -}}
    {{- printf "argocd-secret" -}}
{{- else if and .Values.externalArgoCD.enabled .Values.externalArgoCD.existingSecret -}}
    {{- .Values.externalArgoCD.existingSecret -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper argocd password secret key
*/}}
{{- define "kubegems.argocd.password.secret.key" -}}
{{- if index .Values "argo-cd" "enabled" -}}
    {{- "clearPassword" -}}
{{- else if .Values.externalArgoCD.existingSecretKey -}}
    {{- .Values.externalArgoCD.existingSecretKey -}}
{{- else -}}
    {{- "clearPassword" -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified gitea name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "kubegems.gitea.fullname" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" "gitea" "chartValues" .Values.gitea "context" $) -}}
{{- end -}}

{{/*
Return the proper git address
*/}}
{{- define "kubegems.git.address" -}}
{{- if .Values.gitea.enabled -}}
    {{- printf "http://%s-http:" (include "kubegems.gitea.fullname" .) -}}{{- .Values.gitea.service.http.port -}}
{{- else if .Values.externalGit.enabled -}}
    {{- .Values.externalGit.address -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper git username
*/}}
{{- define "kubegems.git.username" -}}
{{- if .Values.gitea.enabled -}}
    {{- .Values.gitea.gitea.admin.username -}}
{{- else if .Values.externalGit.enabled -}}
    {{- .Values.externalGit.address -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper git password
*/}}
{{- define "kubegems.git.password" -}}
{{- if .Values.gitea.enabled -}}
    {{- .Values.gitea.gitea.admin.password -}}
{{- else if .Values.externalGit.enabled -}}
    {{- .Values.externalGit.password -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper git password secret
*/}}
{{- define "kubegems.git.password.secret" -}}
{{- if and .Values.gitea.enabled -}}
    {{/*gitea do not provide a git password secret*/}}
{{- else if and .Values.externalGit.enabled .Values.externalGit.existingSecret -}}
    {{- .Values.externalGit.existingSecret -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper git password secret key
*/}}
{{- define "kubegems.git.password.secret.key" -}}
{{- if and .Values.gitea.enabled -}}
    {{/*gitea do not provide a git password secret*/}}
{{- else if .Values.externalGit.existingSecretKey -}}
    {{- .Values.externalGit.existingSecretKey -}}
{{- else -}}
    {{- "git-password" -}}
{{- end -}}
{{- end -}}


{{/*
Create a default fully qualified gitea name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "kubegems.chartmuseum.fullname" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" "chartmuseum" "chartValues" .Values.chartmuseum "context" $) -}}
{{- end -}}

{{/*
chartmuseum address
*/}}
{{- define "kubegems.chartmuseum.address" -}}
{{- if and .Values.chartmuseum.enabled -}}
    {{- printf "http://%s.%s:" (include "kubegems.chartmuseum.fullname" .) .Release.Namespace -}}{{- .Values.chartmuseum.service.externalPort -}}
{{- else if .Values.externalAppstore.enabled -}}
    {{- .Values.externalAppstore.address -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper common environment variables
*/}}
{{- define "kubegems.common.env" }}
- name: MICROSERVICE_GATEWAYNAMESPACE
  value: "kubegems-gateway"
- name: MICROSERVICE_ISTIOOPERATORNAME
  value: "kubegems-istio"
- name: JWT_ISSUERADDR
  value: {{ printf "https://%s/api" .Values.ingress.hostname }}
{{- include "kubegems.database.env" . }}
{{- end }}

{{/*
{{ include "kubegems.database.env" . }}
*/}}
{{- define "kubegems.database.env" }}
- name: MYSQL_ADDR
  value: {{ printf "%s:%s" (include "kubegems.database.host" .) (include "kubegems.database.port" .) | quote }}
{{- if (include "kubegems.database.username" .) }}
- name: MYSQL_USERNAME
  value: {{ include "kubegems.database.username" . }}
{{- end }}
{{- if (include "kubegems.database.database" .) }}
- name: MYSQL_DATABASE
  value: {{ include "kubegems.database.database" . }}
{{- end }}
{{- if (include "kubegems.database.password.secret" . ) }}
- name: MYSQL_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ include "kubegems.database.password.secret" . }}
      key: {{ include "kubegems.database.password.secret.key" . }}
{{- end }}
{{/* optional redis config */}}
{{- if (include "kubegems.redis.address" .)  -}}
- name: REDIS_ADDR
  value: {{ include "kubegems.redis.address" . | quote }}
{{- if (include "kubegems.redis.password.secret" . ) }}
- name: REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ include "kubegems.redis.password.secret" . }}
      key: {{ include "kubegems.redis.password.secret.key" . }}
{{- else if (include "kubegems.redis.password" . ) }}
- name: REDIS_PASSWORD
  value: {{ include "kubegems.redis.password" . | quote }}
{{- end -}}
{{- end }}
{{- end }}

{{- define "kubegems.apps.enabled" -}}
{{- (index .Values "kubegems-apps" "enabled") -}}
{{- end -}}

{{/*
Return the proper apps environment variables
{{ include "kubegems.apps.env" . }}
*/}}
{{- define "kubegems.apps.env" }}
{{- if ne (include "kubegems.apps.enabled" .) "false" }}
- name: APPSTORE_ADDR
  value: {{ include "kubegems.chartmuseum.address" . }}
- name: ARGO_ADDR
  value: {{ include "kubegems.argocd.address" . }}
- name: ARGO_USERNAME
  value: {{ include "kubegems.argocd.username" . }}
- name: ARGO_PASSWORD
{{- if (include "kubegems.argocd.password.secret" . ) }}
  valueFrom:
    secretKeyRef:
      name: {{ include "kubegems.argocd.password.secret" . }}
      key: {{ include "kubegems.argocd.password.secret.key" . }}
{{ else }}
  value: {{ include "kubegems.argocd.password" . }}
{{- end -}}
- name: GIT_ADDR
  value: {{ include "kubegems.git.address" . }}
- name: GIT_USERNAME
  value: {{ include "kubegems.git.username" . }}
- name: GIT_PASSWORD
{{- if (include "kubegems.git.password.secret" . ) }}
  valueFrom:
    secretKeyRef:
      name: {{ include "kubegems.git.password.secret" . }}
      key: {{ include "kubegems.git.password.secret.key" . }}
{{ else }}
  value: {{ include "kubegems.git.password" . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Return the proper otel environment variables
{{ include "kubegems.section.env" (dict "name" "kubegems-api" "root" .Values.api ) }}
*/}}
{{- define "kubegems.section.env" }}
- name: KUBEGEMS_DEBUG
  value: {{ ternary "true" "false" (or .root.image.debug ) | quote }}
- name: LOG_LEVEL
  value: {{ .root.logLevel }}
{{- include "kubegems.otel.env" . }}
{{- end }}

{{/*
Return the proper otel environment variables
{{ include "kubegems.otel.env" (dict "name" "kubegems-api" "root" .Values.api ) }}
*/}}
{{- define "kubegems.otel.env" }}
{{- if and .root.opentelemetry .root.opentelemetry.enabled }}
- name: OTEL_ENABLE
  value: "true"
- name: OTEL_K8S_NODE_NAME
  valueFrom:
    fieldRef:
      apiVersion: v1
      fieldPath: spec.nodeName
- name: OTEL_K8S_POD_NAME
  valueFrom:
    fieldRef:
      apiVersion: v1
      fieldPath: metadata.name
- name: OTEL_SERVICE_NAME
  value: {{ .name }}
- name: OTEL_K8S_NAMESPACE
  valueFrom:
    fieldRef:
      apiVersion: v1
      fieldPath: metadata.namespace
- name: OTEL_RESOURCE_ATTRIBUTES
  value: service.name=$(OTEL_SERVICE_NAME),namespace=$(OTEL_K8S_NAMESPACE),node=$(OTEL_K8S_NODE_NAME),pod=$(OTEL_K8S_POD_NAME)
- name: OTEL_EXPORTER_OTLP_ENDPOINT
  value: {{ .root.opentelemetry.endpoint }}
- name: OTEL_EXPORTER_OTLP_INSECURE
  value: "true"
{{- end }}
{{- end }}
