{{- define "consul_creds" -}}
- name: CONSUL_CACERT
  value: "/consul/ca.pem"
- name: CONSUL_HTTP_TOKEN
  valueFrom:
    secretKeyRef:
      name: kubernetes-consul-template
      key: token
- name: CONSUL_HTTP_ADDR
  valueFrom:
    configMapKeyRef:
      name: consul
      key: consul_http_addr
{{- end -}}
