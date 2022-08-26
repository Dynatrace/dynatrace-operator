package types

import (
	"github.com/Dynatrace/dynatrace-operator/src/builder"
	"github.com/Dynatrace/dynatrace-operator/src/builder/api"
	appsv1 "k8s.io/api/apps/v1"
)

type Data = appsv1.StatefulSet
type Modifier = api.Modifier[Data]
type Builder = builder.Builder[Data]
