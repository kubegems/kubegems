{{- define "kubegems.initContainers.database" -}}
- name: init
  image: {{ include "kubegems.init.image" . }}
  imagePullPolicy: {{ .Values.init.image.pullPolicy }}
  args:
  - service
  - migrate
  {{- if .Values.init.migrateModels }}
  - --migratemodels
  {{- end }}
  {{- if .Values.init.initData }}
  - --initdata
  {{- end }}
  {{- if index .Values "kubegems-local" "enabled" }}
  - --globalvalues={{ .Values.global | toJson }}
  {{- end }}
  env:
  {{- include "kubegems.database.env" . | indent 2 }}
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