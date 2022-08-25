package types

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder/api"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Modifier = api.Modifier[v1.ObjectMeta]
type Builder = builder.Builder[v1.ObjectMeta]
