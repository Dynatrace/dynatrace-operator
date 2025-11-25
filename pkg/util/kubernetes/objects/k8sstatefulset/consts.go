package k8sstatefulset

import "github.com/Dynatrace/dynatrace-operator/pkg/api"

const (
	AnnotationPVCHash = api.InternalFlagPrefix + "pvc-hash"
)
