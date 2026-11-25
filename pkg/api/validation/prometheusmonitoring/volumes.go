// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorConflictingVolumeName      = `The PrometheusMonitoring's %s specification uses a volume name that conflicts with an internally managed volume: %s. Please use a different name.`
	errorInvalidVolumeMount         = `The PrometheusMonitoring's %s specification has a volume mount referencing a volume that is not defined in its volumes: %s.`
	errorInvalidVolumeMountPath     = `The PrometheusMonitoring's %s specification has an invalid volume mount path: %s.`
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
		{"targetAllocator", pm.Spec.TargetAllocator.PodSpec, prometheusmonitoring.TargetAllocatorVolumeNames, prometheusmonitoring.TargetAllocatorVolumeMountPaths},
		{"scraper", pm.Spec.Scraper.PodSpec, prometheusmonitoring.ScraperVolumeNames, prometheusmonitoring.ScraperVolumeMountPaths},
		{"gateway", pm.Spec.Gateway.PodSpec, prometheusmonitoring.GatewayVolumeNames, prometheusmonitoring.GatewayVolumeMountPaths},
	}

	for _, c := range components {
		if msg := conflictingVolumeNames(c); msg != "" {
			return msg
		}

		if msg := conflictingVolumeMounts(c); msg != "" {
			return msg
		}
	}

	return ""
}

func conflictingVolumeNames(c reservedVolumes) string {
	for _, volume := range c.spec.Volumes {
		if slices.Contains(c.names, volume.Name) {
			return fmt.Sprintf(errorConflictingVolumeName, c.component, volume.Name)
		}
	}

	return ""
}

func conflictingVolumeMounts(c reservedVolumes) string {
	for _, mount := range c.spec.VolumeMounts {
		if !slices.ContainsFunc(c.spec.Volumes, func(volume types.Volume) bool { return volume.Name == mount.Name }) {
			return fmt.Sprintf(errorInvalidVolumeMount, c.component, mount.Name)
		}

		for _, reserved := range c.mountPath {
			rel, err := filepath.Rel(reserved, mount.MountPath)
			if err != nil {
				return fmt.Sprintf(errorInvalidVolumeMountPath, c.component, mount.MountPath)
			}

			if !strings.HasPrefix(rel, "../") {
				return fmt.Sprintf(errorConflictingVolumeMountPath, c.component, mount.MountPath)
			}
		}
	}

	return ""
}
