package types

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder/api"
	corev1 "k8s.io/api/core/v1"
)

type Modifier = api.Modifier[corev1.PodTemplateSpec]
type Builder = builder.Builder[corev1.PodTemplateSpec]
