package validation

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
)

const (
	warningPreview = `%s mode is in PREVIEW. Please be aware that it is NOT production ready and you may run into bugs.`
)

func previewWarning(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.CloudNativeFullstackMode() {
		log.Info("DynaKube with cloudNativeFullStack was applied, warning was provided.")
		return fmt.Sprintf(warningPreview, "cloudNativeFullStack")
	} else if dynakube.ApplicationMonitoringMode() && dynakube.NeedsCSIDriver() {
		log.Info("DynaKube with applicationMonitoring was applied, warning was provided.")
		return fmt.Sprintf(warningPreview, "applicationMonitoring")
	}
	return ""
}
