FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=dynatrace-operator
LABEL operators.operatorframework.io.bundle.channels.v1=alpha
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.5.0
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v3
LABEL operators.operatorframework.io.bundle.channel.default.v1=alpha

# Copy files to locations specified by labels.
COPY 0.2.1/manifests /manifests/
COPY 0.2.1/metadata /metadata/
LABEL com.redhat.openshift.versions="v4.5,v4.6,v4.7"
LABEL com.redhat.delivery.operator.bundle=true
LABEL com.redhat.delivery.backport=true
