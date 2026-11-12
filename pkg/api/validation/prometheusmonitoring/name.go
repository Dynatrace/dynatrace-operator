// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	dtvalidation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorNameTooLong = `The length limit for the name of a PrometheusMonitoring is %d, because it is the base for the name of resources related to the PrometheusMonitoring. (example: pmName-gateway).
    The limit is necessary because kubernetes uses the name of some resources (example: StatefulSet) for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)`
)

const (
	podTemplateHashLen = 10
	hyphenLen          = 1
	maxPodNameLen      = 63

	// Maximum name length of a StatefulSet before Pods can no longer be created due to too long DNS-1035 label value of `controller-revision-hash` which is added automatically.
	maxStatefulSetNameLength = maxPodNameLen - podTemplateHashLen - hyphenLen

	// The gateway is the only StatefulSet, and its name has the longest suffix of all StatefulSets and Deployments (which allow longer names).
	maxNameLength = maxStatefulSetNameLength - len(prometheusmonitoring.GatewayNameSuffix)
)

func errorNoDNS1035Label() string {
	return dtvalidation.DNS1035LabelViolation("The PrometheusMonitoring name")
}

func nameInvalid(_ context.Context, _ client.Reader, pm *prometheusmonitoring.PrometheusMonitoring) string {
	// Always check the length before DNS1035 validation to prevent false positives due to too long name.
	if len(pm.Name) > maxNameLength {
		return fmt.Sprintf(errorNameTooLong, maxNameLength)
	}

	if errs := validation.IsDNS1035Label(pm.Name); len(errs) != 0 {
		return errorNoDNS1035Label()
	}

	return ""
}
