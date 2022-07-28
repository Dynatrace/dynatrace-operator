package troubleshoot

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretFieldName  = ".dockerconfigjson"
	pullSecretFieldValue = "top-secret"
)

func checkDynakube(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dynakube  ] ")

	logNewTestf("checking if '%s:%s' Dynakube is configured correctly", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)

	tests := []troubleshootFunc{
		checkDynakubeCrdExists,
		checkSelectedDynakubeExists,
		checkApiUrl,
		evaluateDynatraceApiSecretName,
		checkIfDynatraceApiSecretWithTheGivenNameExists,
		checkIfDynatraceApiSecretHasApiToken,
		checkIfDynatraceApiSecretHasPaasToken,
		checkCustomPullSecret,
		checkIfCustomPullSecretWithTheGivenNameExists,
		checkCustomPullSecretHasRequiredTokens,
		checkProxySecret,
		checkIfProxySecretWithTheGivenNameExists,
		checkProxySecretHasRequiredTokens,
	}

	for _, test := range tests {
		if err := test(troubleshootCtx); err != nil {
			logErrorf("'%s:%s' Dynakube isn't valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
			return err
		}
	}

	logOkf("'%s:%s' Dynakube is valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func checkDynakubeCrdExists(troubleshootCtx *troubleshootContext) error {
	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	if err := troubleshootCtx.apiReader.List(context.TODO(), dynakubeList, &client.ListOptions{Namespace: troubleshootCtx.namespaceName}); err != nil {
		logWithErrorf(err, "CRD for Dynakube missing")
		return err
	}
	logInfof("CRD for Dynakube exists")
	return nil
}

func checkSelectedDynakubeExists(troubleshootCtx *troubleshootContext) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}
	logInfof("using '%s:%s' Dynakube", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func checkApiUrl(troubleshootCtx *troubleshootContext) error {
	logInfof("checking if api url is valid")

	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	validation.SetLogger(log)
	if validation.NoApiUrl(nil, &dynakube) != "" {
		logErrorf("api url is invalid")
		return fmt.Errorf("api url is invalid")
	}
	if validation.IsInvalidApiUrl(nil, &dynakube) != "" {
		logErrorf("api url is invalid")
		return fmt.Errorf("api url is invalid")
	}

	logInfof("api url is valid")
	return nil
}

func evaluateDynatraceApiSecretName(troubleshootCtx *troubleshootContext) error {
	logInfof("checking if secret is valid")

	// use dynakube name or tokens value if set
	troubleshootCtx.dynatraceApiSecretName = troubleshootCtx.dynakubeName

	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	if dynakube.Spec.Tokens != "" {
		troubleshootCtx.dynatraceApiSecretName = dynakube.Spec.Tokens
	}
	return nil
}

func checkIfDynatraceApiSecretWithTheGivenNameExists(troubleshootCtx *troubleshootContext) error {
	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynatraceApiSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		logWithErrorf(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return err
	}
	logInfof("'%s:%s' secret exists", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
	return nil
}

func checkIfDynatraceApiSecretHasApiToken(troubleshootCtx *troubleshootContext) error {
	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynatraceApiSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		logWithErrorf(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return err
	}

	apiTokenByte, ok := secret.Data["apiToken"]
	if !ok {
		logErrorf("token apiToken does not exist in '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return fmt.Errorf("token apiToken does not exist in secret")
	}

	apiToken := string(apiTokenByte)
	if apiToken == "" {
		return fmt.Errorf("token apiToken does not exist in secret")
	}

	logInfof("secret token 'apiToken' exists")
	return nil
}

func checkIfDynatraceApiSecretHasPaasToken(troubleshootCtx *troubleshootContext) error {
	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynatraceApiSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		logWithErrorf(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return err
	}

	paasTokenByte, ok := secret.Data["paasToken"]
	if !ok {
		logErrorf("token paasToken does not exist in '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return fmt.Errorf("token paasToken does not exist in secret")
	}

	paasToken := string(paasTokenByte)
	if paasToken == "" {
		logInfof("token paasToken does not exist in secret")
	} else {
		logInfof("secret token 'paasToken' exists")
	}
	return nil
}

func checkCustomPullSecret(troubleshootCtx *troubleshootContext) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	if dynakube.Spec.CustomPullSecret == "" {
		logInfof("customPullSecret not used")
		return nil
	}

	troubleshootCtx.customPullSecretName = dynakube.Spec.CustomPullSecret
	logInfof("'%s:%s' custom pull secret is used", troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
	return nil
}

func checkIfCustomPullSecretWithTheGivenNameExists(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.customPullSecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.customPullSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		logWithErrorf(err, "'%s:%s' custom pull secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
		return err
	}

	logInfof("custom pull secret '%s:%s' exists", troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
	return nil
}

func checkCustomPullSecretHasRequiredTokens(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.customPullSecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.customPullSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		logWithErrorf(err, "'%s:%s' custom pull secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
		return err
	}

	_, hasConfig := secret.Data[pullSecretFieldName]
	if !hasConfig {
		logErrorf("token '%s' does not exist in '%s:%s' secret", pullSecretFieldName, troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
		return fmt.Errorf("token '%s' does not exist in '%s:%s' secret", pullSecretFieldName, troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
	}

	logInfof("secret token '%s' exists", pullSecretFieldName)
	return nil
}

func checkProxySecret(troubleshootCtx *troubleshootContext) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	if dynakube.Spec.Proxy == nil || dynakube.Spec.Proxy.ValueFrom == "" {
		logInfof("proxy secret not used")
		return nil
	}

	troubleshootCtx.proxySecretName = dynakube.Spec.Proxy.ValueFrom
	logInfof("'%s:%s' proxy secret is used", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	return nil
}

func checkIfProxySecretWithTheGivenNameExists(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.proxySecretName == "" {
		return nil
	}

	proxySecret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.proxySecretName, Namespace: troubleshootCtx.namespaceName}, &proxySecret); err != nil {
		logWithErrorf(err, "'%s:%s' proxy secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
		return err
	}

	logInfof("custom pull secret '%s:%s' exists", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	return nil
}

func checkProxySecretHasRequiredTokens(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.proxySecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.proxySecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		logWithErrorf(err, "'%s:%s' proxy secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
		return err
	}

	_, hasProxy := secret.Data[dtclient.CustomProxySecretKey]
	if !hasProxy {
		logErrorf("token '%s' does not exist in '%s:%s' secret", dtclient.CustomProxySecretKey, troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
		return fmt.Errorf("token '%s' does not exist in '%s:%s' secret", dtclient.CustomProxySecretKey, troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	}

	logInfof("secret token '%s' exists", dtclient.CustomProxySecretKey)
	return nil
}
