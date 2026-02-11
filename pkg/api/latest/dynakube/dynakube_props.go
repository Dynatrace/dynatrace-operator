package dynakube

import (
	"net/url"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/conversion"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MaxNameLength is the maximum length of a DynaKube's name, we tend to add suffixes to the name to avoid name collisions for resources related to the DynaKube. (example: dkName-activegate-<some-hash>)
	// The limit is necessary because kubernetes uses the name of some resources (ActiveGate StatefulSet) for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
	MaxNameLength = 40

	// PullSecretSuffix is the suffix appended to the DynaKube name to n.
	PullSecretSuffix = "-pull-secret"

	DefaultMinRequestThresholdMinutes = 15
)

var log = logd.Get().WithName("dynakube-v1beta6")

func (dk *DynaKube) FF() *exp.FeatureFlags {
	return exp.NewFlags(dk.Annotations)
}

func (dk *DynaKube) RemovedFields() *conversion.RemovedFields {
	if dk.Annotations == nil {
		// Set this value here instead of in the conversion package to prevent having to use a pointer to map.
		// Pointer to map is the only way to initialize a nil map function argument.
		dk.Annotations = make(map[string]string)
	}

	return conversion.NewRemovedFields(dk.Annotations)
}

// APIURL is a getter for dk.Spec.APIURL.
func (dk *DynaKube) APIURL() string {
	return dk.Spec.APIURL
}

func (dk *DynaKube) Conditions() *[]metav1.Condition { return &dk.Status.Conditions }

// APIURLHost returns the host of dk.Spec.APIURL
// E.g. if the APIURL is set to "https://my-tenant.dynatrace.com/api", it returns "my-tenant.dynatrace.com"
// If the URL cannot be parsed, it returns an empty string.
func (dk *DynaKube) APIURLHost() string {
	parsedURL, err := url.Parse(dk.APIURL())
	if err != nil {
		return ""
	}

	return parsedURL.Host
}

// PullSecretName returns the name of the pull secret to be used for immutable images.
func (dk *DynaKube) PullSecretName() string {
	if dk.Spec.CustomPullSecret != "" {
		return dk.Spec.CustomPullSecret
	}

	return dk.Name + PullSecretSuffix
}

// PullSecretNames returns the names of the pull secrets to be used for immutable images.
func (dk *DynaKube) PullSecretNames() []string {
	names := []string{
		dk.Name + PullSecretSuffix,
	}
	if dk.Spec.CustomPullSecret != "" {
		names = append(names, dk.Spec.CustomPullSecret)
	}

	return names
}

func (dk *DynaKube) ImagePullSecretReferences() []corev1.LocalObjectReference {
	pullSecretNames := dk.PullSecretNames()
	imagePullSecrets := make([]corev1.LocalObjectReference, len(pullSecretNames))
	for i, pullSecretName := range pullSecretNames {
		imagePullSecrets[i].Name = pullSecretName
	}

	return imagePullSecrets
}

// Tokens returns the name of the Secret to be used for tokens.
func (dk *DynaKube) Tokens() string {
	if tkns := dk.Spec.Tokens; tkns != "" {
		return tkns
	}

	return dk.Name
}

func (dk *DynaKube) TenantUUID() (string, error) {
	if dk.Status.OneAgent.ConnectionInfo.TenantUUID != "" {
		return dk.Status.OneAgent.ConnectionInfo.TenantUUID, nil
	} else if dk.Status.ActiveGate.ConnectionInfo.TenantUUID != "" {
		return dk.Status.ActiveGate.ConnectionInfo.TenantUUID, nil
	}

	return "", errors.New("tenant UUID not available")
}

func (dk *DynaKube) GetDynatraceAPIRequestThreshold() uint16 {
	if dk.Spec.DynatraceAPIRequestThreshold == nil {
		return DefaultMinRequestThresholdMinutes
	}

	return *dk.Spec.DynatraceAPIRequestThreshold
}

func (dk *DynaKube) APIRequestThreshold() time.Duration {
	return time.Duration(dk.GetDynatraceAPIRequestThreshold()) * time.Minute
}

func (dk *DynaKube) IsTokenScopeVerificationAllowed(timeProvider *timeprovider.Provider) bool {
	return timeProvider.IsOutdated(&dk.Status.DynatraceAPI.LastTokenScopeRequest, dk.APIRequestThreshold())
}

func (dk *DynaKube) IsCodeModulesStatusReady() bool {
	if dk.OneAgent().GetCustomCodeModulesImage() != "" || dk.FF().IsPublicRegistry() {
		if dk.OneAgent().GetCodeModulesImage() == "" {
			return false
		}
	} else if dk.OneAgent().GetCodeModulesVersion() == "" || dk.OneAgent().GetCodeModulesVersion() == string(status.CustomImageVersionSource) {
		return false
	}

	return true
}
