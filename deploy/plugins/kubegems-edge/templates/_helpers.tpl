{{/*
Return the proper image name
{{ include "common.images.image" ( dict "imageRoot" .Values.path.to.the.image "global" $) }}
*/}}
{{- define "kubegems-edge.images.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | toString -}}
{{- if .global.kubegemsVersion }}
    {{- $tag = .global.kubegemsVersion | toString -}}
{{- end }}
{{- if .global }}
    {{- if .global.imageRegistry }}
        {{- $registryName = .global.imageRegistry -}}
    {{- end -}}
{{- end -}}
{{- if $registryName }}
    {{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
    {{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end -}}

{{- define "kubegems-edge.fullname" -}}
{{- printf "%s" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubegems-edge.hub.fullname" -}}
{{- printf "%s-hub" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the proper kubegems-edge.server image name
*/}}
{{- define "kubegems-edge.hub.image" -}}
{{- include "kubegems-edge.images.image" (dict "imageRoot" .Values.hub.image "global" .Values.global) -}}
{{- end -}}

{{/*
Return the agent secretName
*/}}
{{- define "kubegems-edge.hub.secretName" -}}
{{- if .Values.hub.tls.secretName -}}
    {{- .Values.hub.tls.secretName -}}
{{- else -}}
    {{- include "kubegems-edge.hub.fullname" . -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper serviceAccount name
*/}}
{{- define "kubegems-edge.hub.serviceAccountName" -}}
{{- if .Values.hub.serviceAccount.create -}}
    {{ default (include "kubegems-edge.hub.fullname" .) .Values.hub.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.hub.serviceAccount.name }}
{{- end -}}
{{- end -}}


# {{- include "kubegems-edge.hostname" .Values.host -}}
{{- define "kubegems-edge.hostname" -}}
{{ first (regexSplit ":" . -1) }}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubegems-edge.server.fullname" -}}
{{- printf "%s-server" (include "common.names.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the proper kubegems-edge.server image name
*/}}
{{- define "kubegems-edge.server.image" -}}
{{ include "kubegems-edge.images.image" (dict "imageRoot" .Values.server.image "global" .Values.global) }}
{{- end -}}

{{/*
Return the agent secretName
*/}}
{{- define "kubegems-edge.server.secretName" -}}
{{- if .Values.server.tls.secretName -}}
    {{- .Values.server.tls.secretName -}}
{{- else -}}
    {{- include "kubegems-edge.server.fullname" . -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper serviceAccount name
*/}}
{{- define "kubegems-edge.server.serviceAccountName" -}}
{{- if .Values.server.serviceAccount.create -}}
    {{ default (include "kubegems-edge.server.fullname" .) .Values.server.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.server.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Return the proper Docker Image Registry Secret Names
*/}}
{{- define "kubegems-edge.imagePullSecrets" -}}
{{- include "common.images.pullSecrets" (dict "images" (list .Values.server.image .Values.hub.image ) "global" .Values.global) -}}
{{- end -}}
 
{{/*
Validate kubegems api configuration
*/}}
{{- define "kubegems-edge.validateValues.server" -}}
{{- end -}}

{{/*
Compile all warnings into a single message.
*/}}
{{- define "kubegems-edge.validateValues" -}}
{{- $messages := list -}}
{{- $messages := append $messages (include "kubegems-edge.validateValues.server" .) -}}
{{- $messages := without $messages "" -}}
{{- $message := join "\n" $messages -}}

{{- if $message -}}
{{-   printf "\nVALUES VALIDATION:\n%s" $message -}}
{{- end -}}
{{- end -}}

