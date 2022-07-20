package mapper

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	UpdatedViaDynakubeAnnotation = "dynatrace.com/updated-via-operator"
	ErrorConflictingNamespace    = "namespace matches two or more DynaKubes which is unsupported. " +
		"refine the labels on your namespace metadata or DynaKube/CodeModules specification"
)

var (
	log = logger.NewDTLogger().WithName("namespace-mapper")
)
