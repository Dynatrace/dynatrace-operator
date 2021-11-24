package validation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
)

const (
	warningCloudNativeFullStack  = `cloudNativeFullStack mode is in PREVIEW. Please be aware that it is NOT production ready and you may run into bugs.`
	warningApplicationMonitoring = `applicationMonitoring mode is in PREVIEW. Please be aware that it is NOT production ready and you may run into bugs.`
)

func previewWarning(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.CloudNativeFullstackMode() {
		log.Info("Dynakube with cloudNativeFullStack was applied, warning was provided.")
		return warningCloudNativeFullStack
	} else if dynakube.ApplicationMonitoringMode() && dynakube.NeedsCSIDriver() {
		log.Info("Dynakube with applicationMonitoring was applied, warning was provided.")
		return warningApplicationMonitoring
	}
	return ""
}
