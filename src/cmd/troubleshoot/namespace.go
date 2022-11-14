package troubleshoot

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkNamespace(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("namespace")

	logNewTestf("checking namespace '%s'", troubleshootCtx.namespaceName)

	tests := []troubleshootFunc{
		checkNamespaceExists,
		checkDynakubeCrdExists,
	}

	for _, test := range tests {
		err := test(troubleshootCtx)

		if err != nil {
			logErrorf(err.Error())
			return errors.Wrapf(err, "check for namespace %s failed", troubleshootCtx.namespaceName)
		}
	}

	logOkf("using namespace '%s'", troubleshootCtx.namespaceName)
	return nil
}

func checkNamespaceExists(troubleshootCtx *troubleshootContext) error {
	namespace := &corev1.Namespace{}
	err := troubleshootCtx.apiReader.Get(troubleshootCtx.context, client.ObjectKey{Name: troubleshootCtx.namespaceName}, namespace)

	if err != nil {
		return errorWithMessagef(err, "namespace '%s' missing", troubleshootCtx.namespaceName)
	}

	logInfof("namespace '%s' exists", troubleshootCtx.namespaceName)
	return nil
}

func checkDynakubeCrdExists(troubleshootCtx *troubleshootContext) error {
	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	err := troubleshootCtx.apiReader.List(troubleshootCtx.context, dynakubeList, &client.ListOptions{Namespace: troubleshootCtx.namespaceName})

	if runtime.IsNotRegisteredError(err) {
		return errorWithMessagef(err, "CRD for Dynakube missing")
	} else if err != nil {
		return errorWithMessagef(err, "could not list Dynakube")
	}

	logInfof("CRD for Dynakube exists")
	return nil
}
