package troubleshoot

import (
	"context"
	"fmt"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretFieldValue = "top-secret"
)

func checkDynakube(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dynakube  ] ")

	logNewTestf("checking if '%s:%s' Dynakube is configured correctly", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)

	tests := []troubleshootFunc{
		checkDynakubeCrdExists,
		getSelectedDynakubeIfItExists,
		checkApiUrl,
		getDynatraceApiSecretIfItExists,
		checkIfDynatraceApiSecretHasApiToken,
		getPullSecretIfItExists,
		checkPullSecretHasRequiredTokens,
		evaluateProxySecret,
		getProxySecretIfItExists,
		checkProxySecretHasRequiredTokens,
	}

	for _, test := range tests {
		err := test(troubleshootCtx)

		if err != nil {
			logErrorf(err.Error())
			return fmt.Errorf("'%s:%s' Dynakube isn't valid. %s",
				troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName, dynakubeNotValidMessage())
		}
	}

	logOkf("'%s:%s' Dynakube is valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func dynakubeNotValidMessage() string {
	return fmt.Sprintf(
		"Target namespace and dynakube can be changed by providing '--%s <namespace>' or '--%s <dynakube>' parameters.",
		namespaceFlagName, dynakubeFlagName)
}

func checkDynakubeCrdExists(troubleshootCtx *troubleshootContext) error {
	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	err := troubleshootCtx.apiReader.List(context.TODO(), dynakubeList, &client.ListOptions{Namespace: troubleshootCtx.namespaceName})

	if runtime.IsNotRegisteredError(err) {
		return errorWithMessagef(err, "CRD for Dynakube missing")
	} else if err != nil {
		return errorWithMessagef(err, "could not list Dynakube")
	}

	logInfof("CRD for Dynakube exists")
	return nil
}

func getSelectedDynakubeIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewDynakubeQuery(troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(context.TODO())
	dynakube, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakubeName})

	if k8serrors.IsNotFound(err) {
		return errorWithMessagef(err,
			"selected '%s:%s' Dynakube does not exist",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	} else if err != nil {
		return errorWithMessagef(err, "could not get Dynakube '%s:%s'",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	} else {
		troubleshootCtx.dynakube = dynakube
	}

	logInfof("using '%s:%s' Dynakube", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func checkApiUrl(troubleshootCtx *troubleshootContext) error {
	logInfof("checking if api url is valid")

	validation.SetLogger(log)
	if validation.NoApiUrl(nil, &troubleshootCtx.dynakube) != "" {
		return fmt.Errorf("api url is invalid")
	}
	if validation.IsInvalidApiUrl(nil, &troubleshootCtx.dynakube) != "" {
		return fmt.Errorf("api url is invalid")
	}

	logInfof("api url is valid")
	return nil
}

func getDynatraceApiSecretIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(context.TODO(), nil, troubleshootCtx.apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakube.Tokens()})

	if err != nil {
		return errorWithMessagef(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	} else {
		troubleshootCtx.dynatraceApiSecret = secret
	}

	logInfof("'%s:%s' secret exists", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	return nil
}

func checkIfDynatraceApiSecretHasApiToken(troubleshootCtx *troubleshootContext) error {
	apiToken, err := kubeobjects.ExtractToken(&troubleshootCtx.dynatraceApiSecret, dtclient.DynatraceApiToken)
	if err != nil {
		return errorWithMessagef(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	}

	if apiToken == "" {
		return fmt.Errorf("'apiToken' token is empty  in '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	}

	logInfof("secret token 'apiToken' exists")
	return nil
}

func getPullSecretIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(context.TODO(), nil, troubleshootCtx.apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakube.PullSecret()})

	if err != nil {
		return errorWithMessagef(err, "'%s:%s' pull secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.PullSecret())
	} else {
		troubleshootCtx.pullSecret = secret
	}

	logInfof("pull secret '%s:%s' exists", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.PullSecret())
	return nil
}

func checkPullSecretHasRequiredTokens(troubleshootCtx *troubleshootContext) error {
	if _, err := kubeobjects.ExtractToken(&troubleshootCtx.pullSecret, dtpullsecret.DockerConfigJson); err != nil {
		return errorWithMessagef(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.PullSecret())
	}

	logInfof("secret token '%s' exists", dtpullsecret.DockerConfigJson)
	return nil
}

func evaluateProxySecret(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.dynakube.Spec.Proxy == nil {
		logInfof("no proxy is configured")
	} else if troubleshootCtx.dynakube.Spec.Proxy.ValueFrom != "" {
		troubleshootCtx.proxySecretName = troubleshootCtx.dynakube.Spec.Proxy.ValueFrom
		logInfof("'%s:%s' proxy secret is configured to be used", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	} else if troubleshootCtx.dynakube.Spec.Proxy.Value != "" {
		logInfof("proxy value is embedded in dynakube")
	}

	return nil
}

func getProxySecretIfItExists(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.proxySecretName == "" {
		return nil
	}

	query := kubeobjects.NewSecretQuery(context.TODO(), nil, troubleshootCtx.apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.proxySecretName})

	if err != nil {
		return errorWithMessagef(err, "'%s:%s' proxy secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	} else {
		troubleshootCtx.proxySecret = secret
	}

	logInfof("custom pull secret '%s:%s' exists", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	return nil
}

func checkProxySecretHasRequiredTokens(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.proxySecretName == "" {
		return nil
	}

	_, err := kubeobjects.ExtractToken(&troubleshootCtx.proxySecret, dtclient.CustomProxySecretKey)

	if err != nil {
		return errorWithMessagef(err, "invalid '%s:%s' secret, missing key '%s'",
			troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName, dtclient.CustomProxySecretKey)
	}

	logInfof("secret token '%s' exists", dtclient.CustomProxySecretKey)
	return nil
}
