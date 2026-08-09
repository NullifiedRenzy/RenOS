{{- define "renos.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "renos.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "renos.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "renos.labels" -}}
helm.sh/chart: {{ include "renos.chart" . }}
{{ include "renos.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "renos.selectorLabels" -}}
app.kubernetes.io/name: {{ include "renos.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "renos.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "renos.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "renos.imagePullSecret" -}}
{{- if .Values.global.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.global.imagePullSecrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{- define "renos.image" -}}
{{- $registry := .Values.global.imageRegistry | default "" -}}
{{- $repository := .repository -}}
{{- $tag := .tag | default .defaultTag -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry $repository $tag -}}
{{- else -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end -}}
{{- end -}}

{{- define "renos.core.fullname" -}}
{{ include "renos.fullname" . }}-core
{{- end }}

{{- define "renos.hub.fullname" -}}
{{ include "renos.fullname" . }}-hub
{{- end }}

{{- define "renos.agent.fullname" -}}
{{ include "renos.fullname" . }}-agent
{{- end }}

{{- define "renos.postgres.fullname" -}}
{{ include "renos.fullname" . }}-postgres
{{- end }}

{{- define "renos.redis.fullname" -}}
{{ include "renos.fullname" . }}-redis
{{- end }}

{{- define "renos.nats.fullname" -}}
{{ include "renos.fullname" . }}-nats
{{- end }}
