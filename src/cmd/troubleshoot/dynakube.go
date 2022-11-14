package troubleshoot

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretFieldValue = "top-secret"
)

func checkDynakube(troubleshootCtx *troubleshootContext) error {
	log = newSubTestLogger("dynakube")

	logNewTestf("checking if '%s:%s' Dynakube is configured correctly", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)

	tests := []troubleshootFunc{
		checkDynakubeCrdExists,
		getSelectedDynakubeIfItExists,
		checkApiUrl,
		getDynatraceApiSecretIfItExists,
		checkIfDynatraceApiSecretHasApiToken,
		getPullSecretIfItExists,
		checkPullSecretHasRequiredTokens,
		setProxySecretIfItExists,
	}

	for _, test := range tests {
		err := test(troubleshootCtx)

		if err != nil {
			logErrorf(err.Error())
			return errors.Wrapf(err, "'%s:%s' Dynakube isn't valid. %s",
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
	err := troubleshootCtx.apiReader.List(troubleshootCtx.context, dynakubeList, &client.ListOptions{Namespace: troubleshootCtx.namespaceName})

	if runtime.IsNotRegisteredError(err) {
		return errorWithMessagef(err, "CRD for Dynakube missing")
	} else if err != nil {
		return errorWithMessagef(err, "could not list Dynakube")
	}

	logInfof("CRD for Dynakube exists")
	return nil
}

func getSelectedDynakubeIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewDynakubeQuery(troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(troubleshootCtx.context)
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
		return errors.New("api url is invalid")
	}
	if validation.IsInvalidApiUrl(nil, &troubleshootCtx.dynakube) != "" {
		return errors.New("api url is invalid")
	}

	logInfof("api url is valid")
	return nil
}

func getDynatraceApiSecretIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(troubleshootCtx.context, nil, troubleshootCtx.apiReader, log)
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
		return errors.New(fmt.Sprintf("'apiToken' token is empty  in '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens()))
	}

	logInfof("secret token 'apiToken' exists")
	return nil
}

func getPullSecretIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(troubleshootCtx.context, nil, troubleshootCtx.apiReader, log)
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

func setProxySecretIfItExists(troubleshootCtx *troubleshootContext) error {
	if !troubleshootCtx.dynakube.HasProxy() {
		logInfof("no proxy is configured")
		return nil
	} else if troubleshootCtx.dynakube.Spec.Proxy.Value != "" {
		logInfof("proxy value is embedded in the dynakube")
		return setProxyFromValue(troubleshootCtx)
	}

	logInfof("'%s:%s' proxy secret is configured to be used", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Spec.Proxy.ValueFrom)
	return setProxyFromSecret(troubleshootCtx)
}

func setProxyFromValue(troubleshootCtx *troubleshootContext) error {
	err := troubleshootCtx.SetTransportProxy(troubleshootCtx.dynakube.Spec.Proxy.Value)

	if err != nil {
		return errorWithMessagef(err, "error parsing proxy value")
	}

	return nil
}

func setProxyFromSecret(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(troubleshootCtx.context, nil, troubleshootCtx.apiReader, log)
	secret, err := query.Get(types.NamespacedName{
		Namespace: troubleshootCtx.namespaceName,
		Name:      troubleshootCtx.dynakube.Spec.Proxy.ValueFrom})

	if err != nil {
		return errorWithMessagef(err, "'%s:%s' proxy secret is missing",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Spec.Proxy.ValueFrom)
	}

	logInfof("proxy secret '%s:%s' exists",
		troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Spec.Proxy.ValueFrom)

	proxyUrl, err := kubeobjects.ExtractToken(&secret, dtclient.CustomProxySecretKey)
	if err != nil {
		return errorWithMessagef(err, "invalid '%s:%s' secret, missing key '%s'",
			troubleshootCtx.namespaceName, troubleshootCtx.proxySecret.Name, dtclient.CustomProxySecretKey)
	}

	logInfof("secret key '%s' exists", dtclient.CustomProxySecretKey)

	err = troubleshootCtx.SetTransportProxy(proxyUrl)
	if err != nil {
		return errorWithMessagef(err, "error parsing proxy secret value")
	}

	return nil
}
