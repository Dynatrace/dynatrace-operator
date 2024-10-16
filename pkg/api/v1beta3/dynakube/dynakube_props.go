package dynakube

import (
	"net/url"
	"strings"
	"time"

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
)

// ApiUrl is a getter for dk.Spec.APIURL.
func (dk *DynaKube) ApiUrl() string {
	return dk.Spec.APIURL
}

func (dk *DynaKube) Conditions() *[]metav1.Condition { return &dk.Status.Conditions }

// ApiUrlHost returns the host of dk.Spec.APIURL
// E.g. if the APIURL is set to "https://my-tenant.dynatrace.com/api", it returns "my-tenant.dynatrace.com"
// If the URL cannot be parsed, it returns an empty string.
func (dk *DynaKube) ApiUrlHost() string {
	parsedUrl, err := url.Parse(dk.ApiUrl())
	if err != nil {
		return ""
	}

	return parsedUrl.Host
}

// PullSecretName returns the name of the pull secret to be used for immutable images.
func (dk *DynaKube) PullSecretName() string {
	if dk.Spec.CustomPullSecret != "" {
		return dk.Spec.CustomPullSecret
	}

	return dk.Name + PullSecretSuffix
}

// PullSecretsNames returns the names of the pull secrets to be used for immutable images.
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
	imagePullSecrets := make([]corev1.LocalObjectReference, 0)
	for _, pullSecretName := range dk.PullSecretNames() {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: pullSecretName,
		})
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

func (dk *DynaKube) TenantUUIDFromApiUrl() (string, error) {
	return tenantUUID(dk.ApiUrl())
}

func runeIs(wanted rune) func(rune) bool {
	return func(actual rune) bool {
		return actual == wanted
	}
}

func tenantUUID(apiUrl string) (string, error) {
	parsedUrl, err := url.Parse(apiUrl)
	if err != nil {
		return "", errors.WithMessagef(err, "problem parsing tenant id from url %s", apiUrl)
	}

	// Path = "/e/<token>/api" -> ["e",  "<tenant>", "api"]
	subPaths := strings.FieldsFunc(parsedUrl.Path, runeIs('/'))
	if len(subPaths) >= 3 && subPaths[0] == "e" && subPaths[2] == "api" {
		return subPaths[1], nil
	}

	hostnameWithDomains := strings.FieldsFunc(parsedUrl.Hostname(), runeIs('.'))
	if len(hostnameWithDomains) >= 1 {
		return hostnameWithDomains[0], nil
	}

	return "", errors.Errorf("problem getting tenant id from API URL '%s'", apiUrl)
}

func (dk *DynaKube) TenantUUIDFromConnectionInfoStatus() (string, error) {
	if dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID != "" {
		return dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID, nil
	} else if dk.Status.ActiveGate.ConnectionInfo.TenantUUID != "" {
		return dk.Status.ActiveGate.ConnectionInfo.TenantUUID, nil
	}

	return "", errors.New("tenant UUID not available")
}

func (dk *DynaKube) ApiRequestThreshold() time.Duration {
	if dk.Spec.DynatraceApiRequestThreshold < 0 {
		dk.Spec.DynatraceApiRequestThreshold = DefaultMinRequestThresholdMinutes
	}

	return time.Duration(dk.Spec.DynatraceApiRequestThreshold) * time.Minute
}

func (dk *DynaKube) IsTokenScopeVerificationAllowed(timeProvider *timeprovider.Provider) bool {
	return timeProvider.IsOutdated(&dk.Status.DynatraceApi.LastTokenScopeRequest, dk.ApiRequestThreshold())
}
