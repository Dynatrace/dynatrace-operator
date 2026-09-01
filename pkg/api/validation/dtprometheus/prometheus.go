// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	prometheusOperatorGroup = "monitoring.coreos.com"

	warningMissingPrometheusCRDs = "Required Prometheus Operator CRDs are not installed on this cluster: %s. Related functionality will not work until they are installed."
)

type prometheusCRD struct {
	resource string
	version  string
	listKind string
}

var requiredPrometheusCRDs = []prometheusCRD{
	{resource: "servicemonitors", version: "v1", listKind: "ServiceMonitorList"},
	{resource: "podmonitors", version: "v1", listKind: "PodMonitorList"},
	{resource: "probes", version: "v1", listKind: "ProbeList"},
	{resource: "scrapeconfigs", version: "v1alpha1", listKind: "ScrapeConfigList"},
}

func missingPrometheusCRDs(ctx context.Context, apiReader client.Reader) string {
	log := logd.FromContext(ctx)
	missing := []string{}

	for _, crd := range requiredPrometheusCRDs {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{Group: prometheusOperatorGroup, Version: crd.version, Kind: crd.listKind})

		resource := fmt.Sprintf("%s.%s", crd.resource, prometheusOperatorGroup)

		err := apiReader.List(ctx, list)
		switch {
		case err == nil:
			continue
		case meta.IsNoMatchError(err):
			missing = append(missing, resource)
		default:
			log.Debug("failed to check for Prometheus CRD, assuming it is present", "resource", resource, "err", err.Error())
		}
	}

	if len(missing) == 0 {
		return ""
	}

	return fmt.Sprintf(warningMissingPrometheusCRDs, strings.Join(missing, ", "))
}
