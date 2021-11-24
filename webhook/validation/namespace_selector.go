package validation

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/mapper"
)

const errorConflictingNamespaceSelector = `The DynaKube's specification tries to inject into namespaces where another Dynakube already injects into, which is not supported.
Make sure the namespaceSelector doesn't conflict with other Dynakubes namespaceSelector
`

func conflictingNamespaceSelector(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if !dynakube.NeedAppInjection() {
		return ""
	}
	dkMapper := mapper.NewDynakubeMapper(context.TODO(), dv.clt, dv.apiReader, dynakube.Namespace, dynakube, log)
	_, err := dkMapper.MatchingNamespaces()
	if err != nil {
		log.Info("requested dynakube has conflicting namespaceSelector", "name", dynakube.Name, "namespace", dynakube.Namespace)
		return errorConflictingNamespaceSelector
	}
	return ""
}
