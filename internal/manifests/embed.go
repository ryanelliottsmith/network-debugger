package manifests

import _ "embed"

//go:embed rbac.yaml
var RBACYAML string

//go:embed configmap.yaml
var ConfigMapYAML string

//go:embed daemonset-host.yaml
var DaemonSetHostYAML string

//go:embed daemonset-overlay.yaml
var DaemonSetOverlayYAML string
