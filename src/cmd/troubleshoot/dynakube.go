package troubleshoot

import (
	"context"
	"fmt"

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
		evaluateDynatraceApiSecretName,
		getDynatraceApiSecretIfItExists,
		checkIfDynatraceApiSecretHasApiToken,
		evaluatePullSecret,
		getPullSecretIfItExists,
		checkPullSecretHasRequiredTokens,
		evaluateProxySecret,
		getProxySecretIfItExists,
		checkProxySecretHasRequiredTokens,
	}

	for _, test := range tests {
		if err := test(troubleshootCtx); err != nil {
			logErrorf(err.Error())
			return fmt.Errorf("'%s:%s' Dynakube isn't valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		}
	}

	logOkf("'%s:%s' Dynakube is valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func checkDynakubeCrdExists(troubleshootCtx *troubleshootContext) error {
	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	if err := troubleshootCtx.apiReader.List(context.TODO(), dynakubeList, &client.ListOptions{Namespace: troubleshootCtx.namespaceName}); err != nil {
		return errorWithMessagef(err, "CRD for Dynakube missing")
	}
	logInfof("CRD for Dynakube exists")
	return nil
}

func getSelectedDynakubeIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewDynakubeQuery(troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(context.TODO())
	if dynakube, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakubeName}); err != nil {
		return errorWithMessagef(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
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

func evaluateDynatraceApiSecretName(troubleshootCtx *troubleshootContext) error {
	logInfof("checking if secret is valid")

	// use dynakube name or tokens value if set
	troubleshootCtx.dynatraceApiSecretName = troubleshootCtx.dynakubeName

	if troubleshootCtx.dynakube.Spec.Tokens != "" {
		troubleshootCtx.dynatraceApiSecretName = troubleshootCtx.dynakube.Spec.Tokens
	}
	return nil
}

func getDynatraceApiSecretIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(context.TODO(), nil, troubleshootCtx.apiReader, log)
	if secret, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynatraceApiSecretName}); err != nil {
		return errorWithMessagef(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
	} else {
		troubleshootCtx.dynatraceApiSecret = secret
	}
	logInfof("'%s:%s' secret exists", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
	return nil
}

func checkIfDynatraceApiSecretHasApiToken(troubleshootCtx *troubleshootContext) error {
	apiToken, err := kubeobjects.ExtractToken(&troubleshootCtx.dynatraceApiSecret, dtclient.DynatraceApiToken)
	if err != nil {
		return errorWithMessagef(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
	}

	if apiToken == "" {
		return fmt.Errorf("'apiToken' token is empty  in '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
	}

	logInfof("secret token 'apiToken' exists")
	return nil
}

func evaluatePullSecret(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.dynakube.Spec.CustomPullSecret == "" {
		troubleshootCtx.pullSecretName = troubleshootCtx.dynakubeName + pullSecretSuffix
		logInfof("customPullSecret not used")
		return nil
	}

	troubleshootCtx.pullSecretName = troubleshootCtx.dynakube.Spec.CustomPullSecret
	logInfof("'%s:%s' pull secret is used", troubleshootCtx.namespaceName, troubleshootCtx.pullSecretName)
	return nil
}

func getPullSecretIfItExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(context.TODO(), nil, troubleshootCtx.apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.pullSecretName})
	if err != nil {
		return errorWithMessagef(err, "'%s:%s' pull secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.pullSecretName)
	} else {
		troubleshootCtx.pullSecret = secret
	}

	logInfof("pull secret '%s:%s' exists", troubleshootCtx.namespaceName, troubleshootCtx.pullSecretName)
	return nil
}

func checkPullSecretHasRequiredTokens(troubleshootCtx *troubleshootContext) error {
	if _, err := kubeobjects.ExtractToken(&troubleshootCtx.pullSecret, dtpullsecret.DockerConfigJson); err != nil {
		return errorWithMessagef(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.pullSecretName)
	}

	logInfof("secret token '%s' exists", dtpullsecret.DockerConfigJson)
	return nil
}

func evaluateProxySecret(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.dynakube.Spec.Proxy == nil || troubleshootCtx.dynakube.Spec.Proxy.ValueFrom == "" {
		logInfof("proxy secret not used")
		return nil
	}

	troubleshootCtx.proxySecretName = troubleshootCtx.dynakube.Spec.Proxy.ValueFrom
	logInfof("'%s:%s' proxy secret is used", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	return nil
}

func getProxySecretIfItExists(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.proxySecretName == "" {
		return nil
	}

	query := kubeobjects.NewSecretQuery(context.TODO(), nil, troubleshootCtx.apiReader, log)
	if secret, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.proxySecretName}); err != nil {
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

	if _, err := kubeobjects.ExtractToken(&troubleshootCtx.proxySecret, dtclient.CustomProxySecretKey); err != nil {
		return errorWithMessagef(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	}

	logInfof("secret token '%s' exists", dtclient.CustomProxySecretKey)
	return nil
}
