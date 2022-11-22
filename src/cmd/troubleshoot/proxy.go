package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"golang.org/x/net/http/httpproxy"
	"k8s.io/apimachinery/pkg/types"
)

func checkProxySettings(troubleshootCtx *troubleshootContext) error {
	return checkProxySettingsWithLog(troubleshootCtx, newSubTestLogger("proxy"))
}

func checkProxySettingsWithLog(troubleshootCtx *troubleshootContext, logger logr.Logger) error {
	log = logger

	var proxyURL string
	logNewCheckf("Analyzing proxy settings ...")

	proxySettingsAvailable := false
	if troubleshootCtx.dynakube.HasProxy() {
		proxySettingsAvailable = true
		logInfof("Reminder: Proxy settings in the Dynakube do not apply to pulling of pod images. Please set your proxy on accordingly on node level.")
		logWarningf("Proxy settings in the Dynakube are ignored for codeModules images due to technical limitations.")

		var err error
		proxyURL, err = getProxyURL(troubleshootCtx)
		if err != nil {
			logErrorf("Unexpected error when reading proxy settings from Dynakube: %v", err)
			return nil
		}
	}

	if checkEnvironmentProxySettings(proxyURL) {
		proxySettingsAvailable = true
	}

	if !proxySettingsAvailable {
		logOkf("No proxy settings found.")
	}
	return nil
}

func checkEnvironmentProxySettings(proxyURL string) bool {
	envProxy := getEnvProxySettings()
	if envProxy == nil {
		return false
	}

	logInfof("Searching environment for proxy settings ...")
	if envProxy.HTTPProxy != "" {
		logWarningf("HTTP_PROXY is set in environment. This setting will be used by the operator for codeModule image pulls.")
		if proxySettingsDiffer(envProxy.HTTPProxy, proxyURL) {
			logWarningf("Proxy settings in the Dynakube and HTTP_PROXY differ.")
		}
	}
	if envProxy.HTTPSProxy != "" {
		logWarningf("HTTPS_PROXY is set in environment. This setting will be used by the operator for codeModule image pulls.")
		if proxySettingsDiffer(envProxy.HTTPSProxy, proxyURL) {
			logWarningf("Proxy settings in the Dynakube and HTTPS_PROXY differ.")
		}
	}
	return true
}

func proxySettingsDiffer(envProxy, dynakubeProxy string) bool {
	return envProxy != "" && dynakubeProxy != "" && envProxy != dynakubeProxy
}

func getEnvProxySettings() *httpproxy.Config {
	envProxy := httpproxy.FromEnvironment()
	if envProxy.HTTPProxy != "" || envProxy.HTTPSProxy != "" {
		return envProxy
	}
	return nil
}

func applyProxySettings(troubleshootCtx *troubleshootContext) error {
	proxyURL, err := getProxyURL(troubleshootCtx)
	if err != nil {
		return err
	}

	if proxyURL != "" {
		err := troubleshootCtx.SetTransportProxy(proxyURL)
		if err != nil {
			return errors.Wrapf(err, "error parsing proxy value")
		}
	}

	return nil
}

func getProxyURL(troubleshootCtx *troubleshootContext) (string, error) {
	if troubleshootCtx.dynakube.Spec.Proxy == nil {
		return "", nil
	}

	if troubleshootCtx.dynakube.Spec.Proxy.Value != "" {
		return troubleshootCtx.dynakube.Spec.Proxy.Value, nil
	}

	if troubleshootCtx.dynakube.Spec.Proxy.ValueFrom != "" {
		err := setProxySecret(troubleshootCtx)
		if err != nil {
			return "", err
		}

		proxyUrl, err := kubeobjects.ExtractToken(troubleshootCtx.proxySecret, dtclient.CustomProxySecretKey)
		if err != nil {
			return "", errors.Wrapf(err, "failed to extract proxy secret field")
		}
		return proxyUrl, nil
	}
	return "", nil
}

func setProxySecret(troubleshootCtx *troubleshootContext) error {
	if troubleshootCtx.proxySecret != nil {
		return nil
	}

	query := kubeobjects.NewSecretQuery(troubleshootCtx.context, nil, troubleshootCtx.apiReader, log)
	secret, err := query.Get(types.NamespacedName{
		Namespace: troubleshootCtx.namespaceName,
		Name:      troubleshootCtx.dynakube.Spec.Proxy.ValueFrom})

	if err != nil {
		return errors.Wrapf(err, "'%s:%s' proxy secret is missing",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Spec.Proxy.ValueFrom)
	}

	troubleshootCtx.proxySecret = &secret
	logInfof("proxy secret '%s:%s' exists",
		troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Spec.Proxy.ValueFrom)
	return nil
}
