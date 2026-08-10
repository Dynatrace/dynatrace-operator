// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MaxNameLength is the maximum length of a DynaKube's name, we tend to add suffixes to the name to avoid name collisions for resources related to the DynaKube. (example: dkName-activegate-<some-hash>)
	// The limit is necessary because kubernetes uses the name of some resources (ActiveGate StatefulSet) for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
	MaxNameLength = 40

	PullSecretSuffix = "-pull-secret"

	DefaultMinRequestThresholdMinutes = 15
)

func (dk *DynaKube) FF() *exp.FeatureFlags {
	return exp.NewFlags(dk.Annotations, false)
}

// APIURL is a getter for dk.Spec.APIURL.
func (dk *DynaKube) APIURL() string { //nolint:staticcheck
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
