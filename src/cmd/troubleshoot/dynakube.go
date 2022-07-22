package troubleshoot

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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
	tslog.SetPrefix("[dynakube  ] ")

	tslog.NewTestf("checking if '%s:%s' Dynakube is configured correctly", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)

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
			tslog.Errorf("'%s:%s' Dynakube isn't valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
			return err
		}
	}

	tslog.Okf("'%s:%s' Dynakube is valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func checkDynakubeCrdExists(troubleshootCtx *troubleshootContext) error {
	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	if err := troubleshootCtx.apiReader.List(context.TODO(), dynakubeList, &client.ListOptions{Namespace: troubleshootCtx.namespaceName}); err != nil {
		tslog.WithErrorf(err, "CRD for Dynakube missing")
		return err
	}
	tslog.Infof("CRD for Dynakube exists")
	return nil
}

func checkSelectedDynakubeExists(troubleshootCtx *troubleshootContext) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}
	tslog.Infof("using '%s:%s' Dynakube", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func checkApiUrl(troubleshootCtx *troubleshootContext) error {
	tslog.Infof("checking if api url is valid")

	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	apiUrl := dynakube.Spec.APIURL

	if apiUrl == validation.ExampleApiUrl {
		tslog.Errorf("api url is an example url '%s'", apiUrl)
		return fmt.Errorf("api url is an example url")
	}

	if apiUrl == "" {
		tslog.Errorf("requested '%s:%s' dynakube has no api url", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return fmt.Errorf("requested '%s:%s' dynakube has no api url", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	}

	if !strings.HasSuffix(apiUrl, "/api") {
		tslog.Errorf("api url does not end with /api '%s'", apiUrl)
		return fmt.Errorf("api url does not end with /api")
	}

	parsedUrl, err := url.Parse(apiUrl)
	if err != nil {
		tslog.WithErrorf(err, "API URL is not a valid URL")
		return fmt.Errorf("API URL is not a valid URL")
	}

	hostname := parsedUrl.Hostname()
	hostnameWithDomains := strings.FieldsFunc(hostname,
		func(r rune) bool { return r == '.' },
	)

	if len(hostnameWithDomains) < 1 || len(hostnameWithDomains[0]) == 0 {
		tslog.Errorf("invalid '%s' hostname in the api url", hostname)
		return fmt.Errorf("invalid hostname in the api url")
	}

	tslog.Infof("api url correctly ends on '/api'")

	tslog.Infof("api url is valid")
	return nil
}

func evaluateDynatraceApiSecretName(troubleshootCtx *troubleshootContext) error {
	tslog.Infof("checking if secret is valid")

	// use dynakube name or tokens value if set
	troubleshootCtx.dynatraceApiSecretName = troubleshootCtx.dynakubeName

	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
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
		tslog.WithErrorf(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return err
	}
	tslog.Infof("'%s:%s' secret exists", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
	return nil
}

func checkIfDynatraceApiSecretHasApiToken(troubleshootCtx *troubleshootContext) error {
	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynatraceApiSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return err
	}

	apiTokenByte, ok := secret.Data["apiToken"]
	if !ok {
		tslog.Errorf("token apiToken does not exist in '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return fmt.Errorf("token apiToken does not exist in secret")
	}

	apiToken := string(apiTokenByte)
	if apiToken == "" {
		return fmt.Errorf("token apiToken does not exist in secret")
	}

	tslog.Infof("secret token 'apiToken' exists")
	return nil
}

func checkIfDynatraceApiSecretHasPaasToken(troubleshootCtx *troubleshootContext) error {
	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynatraceApiSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return err
	}

	paasTokenByte, ok := secret.Data["paasToken"]
	if !ok {
		tslog.Errorf("token paasToken does not exist in '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynatraceApiSecretName)
		return fmt.Errorf("token paasToken does not exist in secret")
	}

	paasToken := string(paasTokenByte)
	if paasToken == "" {
		tslog.Infof("token paasToken does not exist in secret")
	} else {
		tslog.Infof("secret token 'paasToken' exists")
	}
	return nil
}

func checkCustomPullSecret(troubleshootCtx *troubleshootContext) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	if dynakube.Spec.CustomPullSecret == "" {
		tslog.Infof("customPullSecret not used")
		return nil
	}

	troubleshootCtx.customPullSecretName = dynakube.Spec.CustomPullSecret
	tslog.Infof("'%s:%s' custom pull secret is used", troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
	return nil
}

func checkIfCustomPullSecretWithTheGivenNameExists(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.customPullSecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.customPullSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' custom pull secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
		return err
	}

	tslog.Infof("custom pull secret '%s:%s' exists", troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
	return nil
}

func checkCustomPullSecretHasRequiredTokens(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.customPullSecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.customPullSecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' custom pull secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
		return err
	}

	_, hasConfig := secret.Data[pullSecretFieldName]
	if !hasConfig {
		tslog.Errorf("token '%s' does not exist in '%s:%s' secret", pullSecretFieldName, troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
		return fmt.Errorf("token '%s' does not exist in '%s:%s' secret", pullSecretFieldName, troubleshootCtx.namespaceName, troubleshootCtx.customPullSecretName)
	}

	tslog.Infof("secret token '%s' exists", pullSecretFieldName)
	return nil
}

func checkProxySecret(troubleshootCtx *troubleshootContext) error {
	dynakube := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dynakube); err != nil {
		tslog.WithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
		return err
	}

	if dynakube.Spec.Proxy == nil || dynakube.Spec.Proxy.ValueFrom == "" {
		tslog.Infof("proxy secret not used")
		return nil
	}

	troubleshootCtx.proxySecretName = dynakube.Spec.Proxy.ValueFrom
	tslog.Infof("'%s:%s' proxy secret is used", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	return nil
}

func checkIfProxySecretWithTheGivenNameExists(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.proxySecretName == "" {
		return nil
	}

	proxySecret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.proxySecretName, Namespace: troubleshootCtx.namespaceName}, &proxySecret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' proxy secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
		return err
	}

	tslog.Infof("custom pull secret '%s:%s' exists", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	return nil
}

func checkProxySecretHasRequiredTokens(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.proxySecretName == "" {
		return nil
	}

	secret := corev1.Secret{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.proxySecretName, Namespace: troubleshootCtx.namespaceName}, &secret); err != nil {
		tslog.WithErrorf(err, "'%s:%s' proxy secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
		return err
	}

	_, hasProxy := secret.Data[dtclient.CustomProxySecretKey]
	if !hasProxy {
		tslog.Errorf("token '%s' does not exist in '%s:%s' secret", dtclient.CustomProxySecretKey, troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
		return fmt.Errorf("token '%s' does not exist in '%s:%s' secret", dtclient.CustomProxySecretKey, troubleshootCtx.namespaceName, troubleshootCtx.proxySecretName)
	}

	tslog.Infof("secret token '%s' exists", dtclient.CustomProxySecretKey)
	return nil
}
