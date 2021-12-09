package validation

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
)

const (
	errorConflictingNamespaceSelector = `The DynaKube's specification tries to inject into namespaces where another Dynakube already injects into, which is not supported.
Make sure the namespaceSelector doesn't conflict with other Dynakubes namespaceSelector
`
	errorConflictingNamespaceSelectorNoSelector = `The DynaKube does not specificy namespaces where it should inject into while another Dynakube already injects into namespaces, which is not supported.
Make sure you have a namespaceSelector doesn't conflict with other Dynakubes namespaceSelector
`
)

func conflictingNamespaceSelector(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if !dynakube.NeedAppInjection() {
		return ""
	}
	dkMapper := mapper.NewDynakubeMapper(context.TODO(), dv.clt, dv.apiReader, dynakube.Namespace, dynakube)
	_, err := dkMapper.MatchingNamespaces()
	if err != nil && err.Error() == mapper.ErrorConflictingNamespace {
		if dynakube.NamespaceSelector().MatchExpressions == nil && dynakube.NamespaceSelector().MatchLabels == nil {
			log.Info("requested dynakube has conflicting namespaceSelector", "name", dynakube.Name, "namespace", dynakube.Namespace)
			return errorConflictingNamespaceSelectorNoSelector
		} else {
			log.Info("requested dynakube has conflicting namespaceSelector", "name", dynakube.Name, "namespace", dynakube.Namespace)
			return errorConflictingNamespaceSelector
		}

	}
	return ""
}
