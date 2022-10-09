{{- define "kubegems.initContainers.database" -}}
- name: init
  image: {{ include "kubegems.init.image" . }}
  imagePullPolicy: {{ .Values.init.image.pullPolicy }}
  args:
  - service
  - migrate
  env:
  {{- include "kubegems.common.env" . | indent 2 }}
  envFrom:
  - secretRef:
      name: {{ template "kubegems.secret.fullname" . }}
  {{- if .Values.init.extraEnvVarsCM }}
  - configMapRef:
      name: {{ include "common.tplvalues.render" (dict "value" .Values.init.extraEnvVarsCM "context" $) }}
  {{- end }}
  {{- if .Values.init.extraEnvVarsSecret }}
  - secretRef:
      name: {{ include "common.tplvalues.render" (dict "value" .Values.init.extraEnvVarsSecret "context" $) }}
  {{- end }}
  {{- if .Values.init.resources }}
  resources: {{- toYaml .Values.init.resources | nindent 4 }}
  {{- end }}
{{- end -}}