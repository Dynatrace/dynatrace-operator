package troubleshoot

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"golang.org/x/net/http/httpproxy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkProxySettings(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube) error {
	log := baseLog.WithName("proxy")

	var proxyURL string

	logNewCheckf(log, "Analyzing proxy settings ...")

	proxySettingsAvailable := false
	if dk.HasProxy() {
		proxySettingsAvailable = true

		logInfof(log, "Reminder: Proxy settings in the Dynakube do not apply to pulling of pod images. Please set your proxy on accordingly on node level.")
		logWarningf(log, "Proxy settings in the Dynakube are ignored for codeModules images due to technical limitations.")

		var err error

		proxyURL, err = getProxyURL(ctx, apiReader, dk)
		if err != nil {
			logErrorf(log, "Unexpected error when reading proxy settings from Dynakube: %v", err)

			return nil
		}
	}

	if checkEnvironmentProxySettings(log, proxyURL) {
		proxySettingsAvailable = true
	}

	if !proxySettingsAvailable {
		logOkf(log, "No proxy settings found.")
	}

	return nil
}

func checkEnvironmentProxySettings(log logd.Logger, proxyURL string) bool {
	envProxy := getEnvProxySettings()
	if envProxy == nil {
		return false
	}

	logInfof(log, "Searching environment for proxy settings ...")

	if envProxy.HTTPProxy != "" {
		logWarningf(log, "HTTP_PROXY is set in environment. This setting will be used by the operator for codeModule image pulls.")

		if proxySettingsDiffer(envProxy.HTTPProxy, proxyURL) {
			logWarningf(log, "Proxy settings in the Dynakube and HTTP_PROXY differ.")
		}
	}

	if envProxy.HTTPSProxy != "" {
		logWarningf(log, "HTTPS_PROXY is set in environment. This setting will be used by the operator for codeModule image pulls.")

		if proxySettingsDiffer(envProxy.HTTPSProxy, proxyURL) {
			logWarningf(log, "Proxy settings in the Dynakube and HTTPS_PROXY differ.")
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

func getProxyURL(ctx context.Context, apiReader client.Reader, dk *dynakube.DynaKube) (string, error) {
	if !dk.HasProxy() {
		return "", nil
	}

	return dk.Proxy(ctx, apiReader)
}
