{{- if .Values.DOMAIN_URL }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "common.fullname" . }}-environment
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
    higress.io/resource-definer: higress
    app: w7-sitemanager-site-manager-nginx
    group: w7-sitemanager
    w7.cc/group-name: w7-sitemanager
    w7.cc/group-names: {{ .Release.Name }}
  annotations:
    kubernetes.io/ingress.class: higress
    higress.io/resource-definer: higress
    {{- if .Values.ingressForceHttps }}
    higress.io/ssl-redirect: "false"
    w7.cc/ssl-redirect: "false"
    cert-manager.io/cluster-issuer: w7-letsencrypt-prod
    cert-manager.io/renew-before: 30m
    {{- end }}
spec:
  {{- if .Values.ingressForceHttps }}
  tls:
    - hosts: [{{ .Values.DOMAIN_URL | quote }}]
      secretName: {{ printf "%s-tls-secret" .Values.DOMAIN_URL }}
  {{- end }}
  rules:
    - host: {{ .Values.DOMAIN_URL | quote }}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: w7-sitemanager-site-manager-nginx
                port:
                  number: 80
{{- end }}
