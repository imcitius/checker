{{/*
Expand the name of the chart.
*/}}
{{- define "checker-edge.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "checker-edge.fullname" -}}
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

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "checker-edge.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "checker-edge.labels" -}}
helm.sh/chart: {{ include "checker-edge.chart" . }}
{{ include "checker-edge.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "checker-edge.selectorLabels" -}}
app.kubernetes.io/name: {{ include "checker-edge.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "checker-edge.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "checker-edge.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the name of the secret containing the API key.
If existingSecret.name is set, use that; otherwise use the chart-managed secret name.
*/}}
{{- define "checker-edge.apiKeySecretName" -}}
{{- if .Values.existingSecret.name }}
{{- .Values.existingSecret.name }}
{{- else }}
{{- include "checker-edge.fullname" . }}
{{- end }}
{{- end }}

{{/*
Return the key within the secret that holds the API key.
*/}}
{{- define "checker-edge.apiKeySecretKey" -}}
{{- if .Values.existingSecret.name }}
{{- .Values.existingSecret.key }}
{{- else }}
{{- "api-key" }}
{{- end }}
{{- end }}
