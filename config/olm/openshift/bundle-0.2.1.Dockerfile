FROM scratch

# Core 0.2.1 labels.
LABEL operators.operatorframework.io.bundle.channel.default.v1=alpha
LABEL operators.operatorframework.io.0.2.1.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.0.2.1.manifests.v1=manifests/
LABEL operators.operatorframework.io.0.2.1.metadata.v1=metadata/
LABEL operators.operatorframework.io.0.2.1.package.v1=dynatrace-operator
LABEL operators.operatorframework.io.0.2.1.channels.v1=alpha
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.8.0+git
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v3

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# Copy files to locations specified by labels.
COPY 0.2.1/manifests /manifests/
COPY 0.2.1/metadata /metadata/
LABEL com.redhat.openshift.versions="v4.5,v4.6,v4.7"
LABEL com.redhat.delivery.operator.bundle=true
LABEL com.redhat.delivery.backport=true
