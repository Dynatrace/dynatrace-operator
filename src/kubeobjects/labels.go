package kubeobjects

import "github.com/Dynatrace/dynatrace-operator/src/version"

const (
	AppNameLabel      = "app.kubernetes.io/name"
	AppCreatedByLabel = "app.kubernetes.io/created-by"
	AppComponentLabel = "app.kubernetes.io/component"
	AppVersionLabel   = "app.kubernetes.io/version"
	FeatureLabel      = "operator.dynatrace.com/feature"
)

func MergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}

func CommonLabels(dynakubeName, componentName string) map[string]string {
	return map[string]string{
		AppNameLabel:      version.AppName,
		AppComponentLabel: componentName,
		AppCreatedByLabel: dynakubeName,
		AppVersionLabel:   version.Version,
	}
}
