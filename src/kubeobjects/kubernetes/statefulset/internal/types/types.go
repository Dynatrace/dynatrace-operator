package types

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder/api"
	appsv1 "k8s.io/api/apps/v1"
)

type Modifier = api.Modifier[appsv1.StatefulSet]
type Builder = builder.Builder[appsv1.StatefulSet]
