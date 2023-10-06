package kubeobjects

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/version"
)

func NewAnnotations(appName, name, component, ver string) *AppLabels {
	return &AppLabels{
		appMatchLabels: appMatchLabels{
			Name:      appName,
			CreatedBy: name,
			ManagedBy: version.AppName,
		},
		Component: strings.ReplaceAll(component, "_", ""),
		Version:   ver,
	}
}
