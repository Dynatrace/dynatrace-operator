package troubleshoot

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretFieldName  = ".dockerconfigjson"
	pullSecretFieldValue = "top-secret"
)

func checkDynakube(apiReader client.Reader, troubleshootContext *TestData) error {
	tslog.SetPrefix("[dynakube  ] ")

	tslog.NewTestf("checking if '%s:%s' Dynakube is configured correctly ...", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)

	tests := []TestFunc{
		checkDynakubeCrdExists,
		checkSelectedDynakubeExists,
		checkApiUrl,
		checkDynatraceApiSecret,
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
		if err := test(apiReader, troubleshootContext); err != nil {
			tslog.Errorf("'%s:%s' Dynakube isn't valid", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
			return err
		}
	}

	tslog.Okf("'%s:%s' Dynakube is valid", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
	return nil
}

func checkDynakubeCrdExists(apiReader client.Reader, troubleshootContext *TestData) error {
	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	if err := apiReader.List(context.TODO(), dynakubeList, &client.ListOptions{Namespace: troubleshootContext.namespaceName}); err != nil {
		tslog.WithErrorf(err, "CRD for Dynakube missing")
		return err
	}
	tslog.Infof("CRD for Dynakube exists")
	return nil
}

func checkSelectedDynakubeExists(apiReader client.Reader, troubleshootContext *TestData) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return err
	}
	tslog.Infof("using '%s:%s' Dynakube", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
	return nil
}

func checkApiUrl(apiReader client.Reader, troubleshootContext *TestData) error {
	tslog.Infof("checking if api url is valid...")

	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return err
	}

	apiUrl := dynakube.Spec.APIURL

	if len(apiUrl) < 4 || apiUrl[len(apiUrl)-4:] != "/api" {
		tslog.Errorf("api url has to end on '/api' (%s)", apiUrl)
		return fmt.Errorf("api url has to end on '/api' (%s)", apiUrl)
	}

	tslog.Infof("api url correctly ends on '/api'")

	tslog.Infof("api url is valid")
	return nil
}

func checkDynatraceApiSecret(apiReader client.Reader, troubleshootContext *TestData) error {
	tslog.Infof("checking if secret is valid ...")

	// use dynakube name or tokens value if set
	troubleshootContext.dynatraceApiSecretName = troubleshootContext.dynakubeName

	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return err
	}

	if dynakube.Spec.Tokens != "" {
		troubleshootContext.dynatraceApiSecretName = dynakube.Spec.Tokens
	}
	return nil
}

func checkIfDynatraceApiSecretWithTheGivenNameExists(apiReader client.Reader, troubleshootContext *TestData) error {
	secret := corev1.Secret{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynatraceApiSecretName, Namespace: troubleshootContext.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' secret is missing", troubleshootContext.namespaceName, troubleshootContext.dynatraceApiSecretName)
		return err
	}
	tslog.Infof("'%s:%s' secret exists", troubleshootContext.namespaceName, troubleshootContext.dynatraceApiSecretName)
	return nil
}

func checkIfDynatraceApiSecretHasApiToken(apiReader client.Reader, troubleshootContext *TestData) error {
	secret := corev1.Secret{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynatraceApiSecretName, Namespace: troubleshootContext.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' secret is missing", troubleshootContext.namespaceName, troubleshootContext.dynatraceApiSecretName)
		return err
	}

	apiTokenByte, ok := secret.Data["apiToken"]
	if !ok {
		tslog.Errorf("token apiToken does not exist in '%s:%s' secret", troubleshootContext.namespaceName, troubleshootContext.dynatraceApiSecretName)
		return fmt.Errorf("token apiToken does not exist in secret")
	}

	apiToken := string(apiTokenByte)
	if apiToken == "" {
		return fmt.Errorf("token apiToken does not exist in secret")
	}

	tslog.Infof("secret token 'apiToken' exists")
	return nil
}

func checkIfDynatraceApiSecretHasPaasToken(apiReader client.Reader, troubleshootContext *TestData) error {
	secret := corev1.Secret{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynatraceApiSecretName, Namespace: troubleshootContext.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' secret is missing", troubleshootContext.namespaceName, troubleshootContext.dynatraceApiSecretName)
		return err
	}

	paasTokenByte, ok := secret.Data["paasToken"]
	if !ok {
		tslog.Errorf("token paasToken does not exist in '%s:%s' secret", troubleshootContext.namespaceName, troubleshootContext.dynatraceApiSecretName)
		return fmt.Errorf("token paasToken does not exist in secret")
	}

	paasToken := string(paasTokenByte)
	if paasToken == "" {
		return fmt.Errorf("token paasToken does not exist in secret")
	}

	tslog.Infof("secret token 'paasToken' exists")
	return nil
}

func checkCustomPullSecret(apiReader client.Reader, troubleshootContext *TestData) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return err
	}

	if dynakube.Spec.CustomPullSecret == "" {
		tslog.Infof("customPullSecret not used")
		return nil
	}

	troubleshootContext.customPullSecretName = dynakube.Spec.CustomPullSecret
	tslog.Infof("'%s:%s' custom pull secret is used", troubleshootContext.namespaceName, troubleshootContext.customPullSecretName)
	return nil
}

func checkIfCustomPullSecretWithTheGivenNameExists(apiReader client.Reader, troubleshootContext *TestData) error {
	if troubleshootContext.customPullSecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.customPullSecretName, Namespace: troubleshootContext.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' custom pull secret is missing", troubleshootContext.namespaceName, troubleshootContext.customPullSecretName)
		return err
	}

	tslog.Infof("custom pull secret '%s:%s' exists", troubleshootContext.namespaceName, troubleshootContext.customPullSecretName)
	return nil
}

func checkCustomPullSecretHasRequiredTokens(apiReader client.Reader, troubleshootContext *TestData) error {
	if troubleshootContext.customPullSecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.customPullSecretName, Namespace: troubleshootContext.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' custom pull secret is missing", troubleshootContext.namespaceName, troubleshootContext.customPullSecretName)
		return err
	}

	_, hasConfig := secret.Data[pullSecretFieldName]
	if !hasConfig {
		tslog.Errorf("token '%s' does not exist in '%s:%s' secret", pullSecretFieldName, troubleshootContext.namespaceName, troubleshootContext.customPullSecretName)
		return fmt.Errorf("token '%s' does not exist in '%s:%s' secret", pullSecretFieldName, troubleshootContext.namespaceName, troubleshootContext.customPullSecretName)
	}

	tslog.Infof("secret token '%s' exists", pullSecretFieldName)
	return nil
}

func checkProxySecret(apiReader client.Reader, troubleshootContext *TestData) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootContext.namespaceName, troubleshootContext.dynakubeName)
		return err
	}

	if dynakube.Spec.Proxy == nil || dynakube.Spec.Proxy.ValueFrom == "" {
		tslog.Infof("proxy secret not used")
		return nil
	}

	troubleshootContext.proxySecretName = dynakube.Spec.Proxy.ValueFrom
	tslog.Infof("'%s:%s' proxy secret is used", troubleshootContext.namespaceName, troubleshootContext.proxySecretName)
	return nil
}

func checkIfProxySecretWithTheGivenNameExists(apiReader client.Reader, troubleshootContext *TestData) error {
	if troubleshootContext.proxySecretName == "" {
		return nil
	}

	proxySecret := corev1.Secret{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.proxySecretName, Namespace: troubleshootContext.namespaceName}, &proxySecret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' proxy secret is missing", troubleshootContext.namespaceName, troubleshootContext.proxySecretName)
		return err
	}

	tslog.Infof("custom pull secret '%s:%s' exists", troubleshootContext.namespaceName, troubleshootContext.proxySecretName)
	return nil
}

func checkProxySecretHasRequiredTokens(apiReader client.Reader, troubleshootContext *TestData) error {
	if troubleshootContext.proxySecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.proxySecretName, Namespace: troubleshootContext.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' proxy secret is missing", troubleshootContext.namespaceName, troubleshootContext.proxySecretName)
		return err
	}

	_, hasProxy := secret.Data[dtclient.CustomProxySecretKey]
	if !hasProxy {
		tslog.Errorf("token '%s' does not exist in '%s:%s' secret", dtclient.CustomProxySecretKey, troubleshootContext.namespaceName, troubleshootContext.proxySecretName)
		return fmt.Errorf("token '%s' does not exist in '%s:%s' secret", dtclient.CustomProxySecretKey, troubleshootContext.namespaceName, troubleshootContext.proxySecretName)
	}

	tslog.Infof("secret token '%s' exists", dtclient.CustomProxySecretKey)
	return nil
}
