apiVersion: w7panel.w7.com/v1alpha1
kind: Site
metadata:
  name: {{ .Release.Name }}
  annotations:
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-weight: "-20"
    helm.sh/hook-delete-policy: before-hook-creation
spec:
  host: {{ .Values.DOMAIN_URL }}
  siteIdentifier: __APPLICATION_IDENTIFIER__
  target:
    apiVersion: w7panel.w7.com/v1alpha1
    kind: AppGroup
    name: {{ .Release.Name }}
    namespace: {{ .Release.Namespace }}
