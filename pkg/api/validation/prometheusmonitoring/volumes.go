// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	pmconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/consts"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorConflictingVolumeName      = `The PrometheusMonitoring's %s specification uses a volume name that conflicts with an internally managed volume: %s. Please use a different name.`
	errorConflictingVolumeMountPath = `The PrometheusMonitoring's %s specification uses a volume mount path that conflicts with an internally managed volume: %s. Please use a different path.`
)

type reservedVolumes struct {
	component string
	spec      prometheusmonitoring.PodSpec
	names     []string
	mountPath []string
}

func conflictingVolumes(_ context.Context, _ client.Reader, pm *prometheusmonitoring.PrometheusMonitoring) string {
	components := []reservedVolumes{
		{"targetAllocator", pm.Spec.TargetAllocator.PodSpec, pmconsts.TargetAllocatorVolumeNames, pmconsts.TargetAllocatorVolumeMountPaths},
		{"scraper", pm.Spec.Scraper.PodSpec, pmconsts.ScraperVolumeNames, pmconsts.ScraperVolumeMountPaths},
		{"gateway", pm.Spec.Gateway.PodSpec, pmconsts.GatewayVolumeNames, pmconsts.GatewayVolumeMountPaths},
	}

	for _, c := range components {
		for _, volume := range c.spec.Volumes {
			if slices.Contains(c.names, volume.Name) {
				return fmt.Sprintf(errorConflictingVolumeName, c.component, volume.Name)
			}
		}

		for _, mount := range c.spec.VolumeMounts {
			if slices.Contains(c.mountPath, mount.MountPath) {
				return fmt.Sprintf(errorConflictingVolumeMountPath, c.component, mount.MountPath)
			}
		}
	}

	return ""
}
