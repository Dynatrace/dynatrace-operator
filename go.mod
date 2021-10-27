module github.com/Dynatrace/dynatrace-operator

go 1.16

require (
	github.com/container-storage-interface/spec v1.5.0
	github.com/containers/image/v5 v5.16.1
	github.com/evanphx/json-patch v4.11.0+incompatible
	github.com/go-logr/logr v0.4.0
	github.com/klauspost/compress v1.13.6
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/spf13/afero v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.1
	golang.org/x/sys v0.0.0-20211020174200-9d6173849985
	google.golang.org/grpc v1.41.0
	istio.io/api v0.0.0-20211020081732-2de5b65af1fe
	istio.io/client-go v1.11.4
	k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/controller-runtime v0.10.2
)
