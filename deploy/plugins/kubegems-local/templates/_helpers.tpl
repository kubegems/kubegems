{{/*
Return the proper image name
{{ include "common.images.image" ( dict "imageRoot" .Values.path.to.the.image "global" $) }}
*/}}
{{- define "kubegems.images.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | toString -}}
{{- if or (not $tag) (eq $tag "latest") -}}
    {{- $tag = printf "v%s" .root.Chart.AppVersion | toString -}}
{{- end -}}
{{- if .global.kubegemsVersion }}
    {{- $tag = .global.kubegemsVersion | toString -}}
{{- end }}
{{- if .global.imageRegistry }}
    {{- $registryName = .global.imageRegistry -}}
{{- end -}}
{{- if $registryName }}
    {{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
    {{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubegems-local.agent.fullname" -}}
{{- printf "%s-agent" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the agent secretName
*/}}
{{- define "kubegems-local.agent.secretName" -}}
{{- if .Values.agent.tls.secretName -}}
    {{- .Values.agent.tls.secretName -}}
{{- else }}
    {{- include "kubegems-local.agent.fullname" . -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper kubegems-local.agent image name
*/}}
{{- define "kubegems-local.agent.image" -}}
{{ include "kubegems.images.image" (dict "imageRoot" .Values.agent.image "global" .Values.global "root" .) }}
{{- end -}}

{{/*
Return the proper agent serviceAccount name
*/}}
{{- define "kubegems-local.agent.serviceAccountName" -}}
{{- if .Values.agent.serviceAccount.create -}}
    {{ default (include "kubegems-local.agent.fullname" .) .Values.agent.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.agent.serviceAccount.name }}
{{- end -}}
{{- end -}}


{{/*
Return the proper kubegems-local.kubectl image name
*/}}
{{- define "kubegems-local.kubectl.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.kubectl.image "global" .Values.global) }}
{{- end -}}

{{- define "kubegems-local.kubectl.fullname" -}}
{{- printf "%s-kubectl" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubegems-local.controller.fullname" -}}
{{- printf "%s-controller" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "kubegems-local.controller.webhook.fullname" -}}
{{- printf "%s-webhook" (include "kubegems-local.controller.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the name of the controller service account to use
*/}}
{{- define "kubegems-local.controller.serviceAccountName" -}}
{{- if .Values.controller.serviceAccount.create -}}
    {{ default (include "kubegems-local.controller.fullname" .) .Values.controller.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.controller.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Return the controller webhook secretName
*/}}
{{- define "kubegems-local.controller.webhook.secretName" -}}
{{- if .Values.controller.webhook.secretName -}}
    {{- .Values.controller.webhook.secretName -}}
{{- else }}
    {{- include "kubegems-local.controller.webhook.fullname" . -}}
{{- end -}}
{{- end -}}


{{/*
Return the proper kubegems-local image name
*/}}
{{- define "kubegems-local.controller.image" -}}
{{ include "kubegems.images.image" (dict "imageRoot" .Values.controller.image "global" .Values.global "root" .) }}
{{- end -}}

{{/*
Return the proper Docker Image Registry Secret Names
*/}}
{{- define "kubegems-local.imagePullSecrets" -}}
{{- include "common.images.pullSecrets" (dict "images" (list .Values.agent.image .Values.controller.image) "global" .Values.global) -}}
{{- end -}}


{{/*
Return true if cert-manager required annotations for TLS signed certificates are set in the Ingress annotations
Ref: https://cert-manager.io/docs/usage/ingress/#supported-annotations
*/}}
{{- define "kubegems-local.ingress.certManagerRequest" -}}
{{ if or (hasKey . "cert-manager.io/cluster-issuer") (hasKey . "cert-manager.io/issuer") }}
    {{- true -}}
{{- end -}}
{{- end -}}


{{/*
Validate agent configuration
*/}}
{{- define "kubegems-local.validateValues.agent" -}}
{{- end -}}

{{/*
Compile all warnings into a single message.
*/}}
{{- define "kubegems-local.validateValues" -}}
{{- $messages := list -}}
{{- $messages := append $messages (include "kubegems-local.validateValues.agent" .) -}}
{{- $messages := without $messages "" -}}
{{- $message := join "\n" $messages -}}

{{- if $message -}}
{{-   printf "\nVALUES VALIDATION:\n%s" $message -}}
{{- end -}}
{{- end -}}

