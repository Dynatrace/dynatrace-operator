package mapper

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	UpdatedViaDynakubeAnnotation = "dynatrace.com/updated-via-operator"
	ErrorConflictingNamespace    = "namespace matches two or more DynaKubes which is unsupported. " +
		"refine the labels on your namespace metadata or DynaKube/CodeModules specification"
)

var (
	log = logd.Get().WithName("namespace-mapper")
)
