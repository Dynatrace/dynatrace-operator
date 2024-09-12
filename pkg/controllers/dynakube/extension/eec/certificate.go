package eec

import "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"

func getCertificateAltNames(dkName string) []string {
	return []string{
		dkName + "-extensions-controller.dynatrace",
		dkName + "-extensions-controller.dynatrace.svc",
		dkName + "-extensions-controller.dynatrace.svc.cluster.local",
	}
}

func getTlsSecretName(dkName string) string {
	return dkName + consts.ExtensionsTlsSecretSuffix
}
