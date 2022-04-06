package validation

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/agproxysecret"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorConflictingActiveGateSections = `The DynaKube's specification tries to use the deprecated ActiveGate section(s) alongside the new ActiveGate section, which is not supported.
`

	errorInvalidActiveGateCapability = `The DynaKube's specification tries to use an invalid capability in ActiveGate section, invalid capability=%s.
Make sure you correctly specify the ActiveGate capabilities in your custom resource.
`

	errorDuplicateActiveGateCapability = `The DynaKube's specification tries to specify duplicate capabilities in the ActiveGate section, duplicate capability=%s.
Make sure you don't duplicate an Activegate capability in your custom resource.
`
	warningMissingActiveGateMemoryLimit = `ActiveGate specification missing memory limits. Can cause excess memory usage.`

	errorInvalidActiveGateProxyUrl = `The DynaKube's specification has an invalid Proxy URL value set. Make sure you correctly specify the URL in your custom resource.`
	errorInvalidEvalCharacter      = `The DynaKube's specification has an invalid Proxy password value set. Make sure you correctly escape quotation mark, backtick and backslash characters using backslash.`

	errorMissingActiveGateProxySecret = `The Proxy secret indicated by the DynaKube specification doesn't exist.`

	errorInvalidProxySecretFormat = `The Proxy secret indicated by the DynaKube specification has an invalid format. Make sure you correctly creates the secret.`

	errorInvalidProxySecretUrl           = `The Proxy secret indicated by the DynaKube specification has an invalid URL value set. Make sure you correctly specify the URL in the secret.`
	errorInvalidProxySecretEvalCharacter = `The Proxy secret indicated by the DynaKube specification has an invalid Proxy password value set. Make sure you correctly escape quotation mark, backtick and backslash characters using backslash.`
)

func conflictingActiveGateConfiguration(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.DeprecatedActiveGateMode() && dynakube.ActiveGateMode() {
		log.Info("requested dynakube has conflicting active gate configuration", "name", dynakube.Name, "namespace", dynakube.Namespace)
		return errorConflictingActiveGateSections
	}
	return ""
}

func duplicateActiveGateCapabilities(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ActiveGateMode() {
		capabilities := dynakube.Spec.ActiveGate.Capabilities
		duplicateChecker := map[dynatracev1beta1.CapabilityDisplayName]bool{}
		for _, capability := range capabilities {
			if duplicateChecker[capability] {
				log.Info("requested dynakube has duplicates in the active gate capabilities section", "name", dynakube.Name, "namespace", dynakube.Namespace)
				return fmt.Sprintf(errorDuplicateActiveGateCapability, capability)
			}
			duplicateChecker[capability] = true
		}
	}
	return ""
}

func invalidActiveGateCapabilities(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ActiveGateMode() {
		capabilities := dynakube.Spec.ActiveGate.Capabilities
		for _, capability := range capabilities {
			if _, ok := dynatracev1beta1.ActiveGateDisplayNames[capability]; !ok {
				log.Info("requested dynakube has invalid active gate capability", "name", dynakube.Name, "namespace", dynakube.Namespace)
				return fmt.Sprintf(errorInvalidActiveGateCapability, capability)
			}
		}
	}
	return ""
}

func missingActiveGateMemoryLimit(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ActiveGateMode() {
		if !memoryLimitSet(dynakube.Spec.ActiveGate.Resources) {
			return warningMissingActiveGateMemoryLimit
		}
	}
	return ""
}

func invalidActiveGateProxyUrl(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Spec.Proxy != nil {
		if len(dynakube.Spec.Proxy.ValueFrom) > 0 {
			var proxySecret corev1.Secret
			err := dv.clt.Get(context.TODO(), client.ObjectKey{Name: dynakube.Spec.Proxy.ValueFrom, Namespace: dynakube.Namespace}, &proxySecret)
			if k8serrors.IsNotFound(err) {
				return errorMissingActiveGateProxySecret
			} else if err != nil {
				return fmt.Sprintf("error occurred while reading PROXY secret indicated in the Dynakube specification (%s)", err.Error())
			}
			proxyUrl, ok := proxySecret.Data[agproxysecret.ProxySecretKey]
			if !ok {
				return errorInvalidProxySecretFormat
			}
			return validateProxyUrl(string(proxyUrl), errorInvalidProxySecretUrl, errorInvalidProxySecretEvalCharacter)
		} else if len(dynakube.Spec.Proxy.Value) > 0 {
			return validateProxyUrl(dynakube.Spec.Proxy.Value, errorInvalidActiveGateProxyUrl, errorInvalidEvalCharacter)
		}
	}
	return ""
}

func memoryLimitSet(resources corev1.ResourceRequirements) bool {
	return resources.Limits != nil && resources.Limits.Memory() != nil
}

// proxyUrl is valid if
// 1) encoded
// 2) "`\ are escaped using \
func validateProxyUrl(proxyUrl string, parseErrorMessage string, evalErrorMessage string) string {
	if parsedUrl, err := url.Parse(proxyUrl); err != nil {
		return parseErrorMessage
	} else {
		password, _ := parsedUrl.User.Password()
		if isEvalEscapeNeeded(password) {
			return evalErrorMessage
		}
	}
	return ""
}

// 'eval' command is used by entrypoint.sh:readSecret function to return its result.
// For this reason quotation mark ", backtick ` and backslash \ characters have to be escaped using backslash.
func isEvalEscapeNeeded(str string) bool {
	previousChar := '\000'
	for _, char := range str {
		if char == '"' || char == '`' {
			if previousChar != '\\' {
				return true
			}
			previousChar = char
		} else if previousChar == '\\' {
			if char != '\\' {
				return true
			}
			previousChar = '\000'
		} else {
			previousChar = char
		}
	}
	return false
}
