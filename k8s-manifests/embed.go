package k8s_manifests

import _ "embed"

//go:embed argocd/helm/values.yaml
var ArgoValuesYaml []byte
