module github.com/Dynatrace/dynatrace-operator

go 1.15

require (
	github.com/containers/image/v5 v5.9.0
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.15.0
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	gotest.tools v2.2.0+incompatible
	istio.io/api v0.0.0-20201217173512-1f62aaeb5ee3
	istio.io/client-go v1.8.1
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.4
)
