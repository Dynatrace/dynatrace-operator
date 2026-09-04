// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	prometheusOperatorGroup = "monitoring.coreos.com"

	warningMissingPrometheusCRDs = "Required Prometheus Operator CRDs are not installed on this cluster: %s. Related functionality will not work until they are installed."
)

type prometheusCRD struct {
	schema.GroupVersionKind
	resource string
}

var requiredPrometheusCRDs = []prometheusCRD{
	{GroupVersionKind: schema.GroupVersionKind{Group: prometheusOperatorGroup, Version: "v1", Kind: "ServiceMonitor"}, resource: "servicemonitors"},
	{GroupVersionKind: schema.GroupVersionKind{Group: prometheusOperatorGroup, Version: "v1", Kind: "PodMonitor"}, resource: "podmonitors"},
	{GroupVersionKind: schema.GroupVersionKind{Group: prometheusOperatorGroup, Version: "v1", Kind: "Probe"}, resource: "probes"},
	{GroupVersionKind: schema.GroupVersionKind{Group: prometheusOperatorGroup, Version: "v1alpha1", Kind: "ScrapeConfig"}, resource: "scrapeconfigs"},
}

func missingPrometheusCRDs(ctx context.Context, apiReader client.Reader) string {
	missing := []string{}

	for _, crd := range requiredPrometheusCRDs {
		if k8scrd.IsInstalled(ctx, apiReader, crd.GroupVersionKind) {
			continue
		}

		missing = append(missing, fmt.Sprintf("%s.%s", crd.resource, crd.Group))
	}

	if len(missing) == 0 {
		return ""
	}

	return fmt.Sprintf(warningMissingPrometheusCRDs, strings.Join(missing, ", "))
}
