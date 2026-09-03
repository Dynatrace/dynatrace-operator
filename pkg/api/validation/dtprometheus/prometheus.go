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
	kind     string
}

var requiredPrometheusCRDs = []prometheusCRD{
	{resource: "servicemonitors", version: "v1", kind: "ServiceMonitor"},
	{resource: "podmonitors", version: "v1", kind: "PodMonitor"},
	{resource: "probes", version: "v1", kind: "Probe"},
	{resource: "scrapeconfigs", version: "v1alpha1", kind: "ScrapeConfig"},
}

func missingPrometheusCRDs(ctx context.Context, apiReader client.Reader) string {
	log := logd.FromContext(ctx)
	missing := []string{}

	for _, crd := range requiredPrometheusCRDs {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(schema.GroupVersionKind{Group: prometheusOperatorGroup, Version: crd.version, Kind: crd.kind})

		resource := fmt.Sprintf("%s.%s", crd.resource, prometheusOperatorGroup)

		err := apiReader.Get(ctx, client.ObjectKey{Namespace: "default", Name: "default"}, obj)

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
