package annotations

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
)

func NewAnnotations(appName, name, component, ver string) *labels.AppLabels {
	return &labels.AppLabels{
		AppMatchLabels: labels.AppMatchLabels{
			Name:      appName,
			CreatedBy: name,
			ManagedBy: version.AppName,
		},
		Component: strings.ReplaceAll(component, "_", ""),
		Version:   ver,
	}
}
