module github.com/Dynatrace/dynatrace-operator

go 1.24.2

require (
	github.com/Dynatrace/dynatrace-bootstrapper v1.1.1
	github.com/container-storage-interface/spec v1.11.0
	github.com/docker/cli v28.4.0+incompatible
	github.com/evanphx/json-patch v5.9.11+incompatible
	github.com/go-logr/logr v1.4.3
	github.com/google/go-containerregistry v0.20.6
	github.com/google/uuid v1.6.0
	github.com/klauspost/compress v1.18.0
	github.com/kubernetes-csi/csi-lib-utils v0.22.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.23.2
	github.com/spf13/afero v1.14.0
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component v1.40.0
	go.opentelemetry.io/collector/config/configtls v1.40.0
	go.opentelemetry.io/collector/confmap v1.40.0
	go.opentelemetry.io/collector/pipeline v1.40.0
	go.opentelemetry.io/collector/service v0.134.0
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56
	golang.org/x/mod v0.28.0
	golang.org/x/net v0.43.0
	golang.org/x/oauth2 v0.31.0
	golang.org/x/sys v0.36.0
	google.golang.org/grpc v1.75.0
	gopkg.in/yaml.v3 v3.0.1
	istio.io/api v1.27.1
	istio.io/client-go v1.27.1
	k8s.io/api v0.34.0
	k8s.io/apiextensions-apiserver v0.34.0
	k8s.io/apimachinery v0.34.0
	k8s.io/client-go v0.34.0
	k8s.io/klog/v2 v2.130.1
	k8s.io/kubelet v0.34.0
	k8s.io/mount-utils v0.34.0
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397
	sigs.k8s.io/controller-runtime v0.22.0
	sigs.k8s.io/e2e-framework v0.3.0
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.3 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250323135004-b31fac66206e // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.1 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.2.2 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/otlptranslator v0.0.0-20250717125610-8549f4ab4f8f // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/vbatts/tar-split v0.12.1 // indirect
	github.com/vladimirvivien/gexe v0.2.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.134.0 // indirect
	go.opentelemetry.io/collector/component/componenttest v0.134.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.40.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.134.0 // indirect
	go.opentelemetry.io/collector/connector v0.134.0 // indirect
	go.opentelemetry.io/collector/connector/connectortest v0.134.0 // indirect
	go.opentelemetry.io/collector/connector/xconnector v0.134.0 // indirect
	go.opentelemetry.io/collector/consumer v1.40.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.134.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.134.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.134.0 // indirect
	go.opentelemetry.io/collector/exporter v0.134.0 // indirect
	go.opentelemetry.io/collector/exporter/exportertest v0.134.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.134.0 // indirect
	go.opentelemetry.io/collector/extension v1.40.0 // indirect
	go.opentelemetry.io/collector/extension/extensioncapabilities v0.134.0 // indirect
	go.opentelemetry.io/collector/extension/extensiontest v0.134.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.40.0 // indirect
	go.opentelemetry.io/collector/internal/fanoutconsumer v0.134.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.134.0 // indirect
	go.opentelemetry.io/collector/pdata v1.40.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.134.0 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.134.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.134.0 // indirect
	go.opentelemetry.io/collector/processor v1.40.0 // indirect
	go.opentelemetry.io/collector/processor/processortest v0.134.0 // indirect
	go.opentelemetry.io/collector/processor/xprocessor v0.134.0 // indirect
	go.opentelemetry.io/collector/receiver v1.40.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.134.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.134.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.12.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.17.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.36.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.13.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.13.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.59.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.13.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.37.0 // indirect
	go.opentelemetry.io/otel/log v0.13.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.13.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/term v0.34.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/component-base v0.34.0 // indirect
	k8s.io/kube-openapi v0.0.0-20250710124328-f3f2b991d03b // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
)
