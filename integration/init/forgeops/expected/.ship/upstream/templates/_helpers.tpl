{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{define "name"}}{{default "opendj" .Values.nameOverride | trunc 63 }}{{end}}
{{define "fullname"}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{end}}

{{/* work in progress. TODO reduce dj image boilerplate */}}
{{define "dscontainer"}}
image:  {{ .Values.image.repository }}:{{ .Values.image.tag }}
imagePullPolicy: {{ .Values.image.pullPolicy }}
volumeMounts:
- name: dj-secrets
    mountPath: /var/run/secrets/opendj
- name: db
    mountPath: /opt/opendj/data
envFrom:
- configMapRef:
    name: {{ .Values.instance }}
env:
- name: NAMESPACE
  valueFrom:
       fieldRef:
         fieldPath: metadata.namespace
{{end}}