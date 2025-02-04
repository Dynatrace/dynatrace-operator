package builder

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/builder"
	appsv1 "k8s.io/api/apps/v1"
)

type Data = appsv1.StatefulSet
type Modifier = builder.Modifier[Data]
type Builder = builder.GenericBuilder[Data]

func NewBuilder(data Data) Builder {
	return builder.NewBuilder(data)
}
