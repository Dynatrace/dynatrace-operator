package builder

import (
	builder "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder/generic"
	appsv1 "k8s.io/api/apps/v1"
)

type Data = appsv1.StatefulSet
type Modifier = builder.Modifier[Data]
type Builder = builder.GenericBuilder[Data]

func NewBuilder(data Data) Builder {
	return builder.NewBuilder(data)
}
