// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	k8sversion "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestInvalidActiveGateCapabilities(t *testing.T) {
	t.Run("conflicting dynakube specs", func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorInvalidActiveGateCapability, "invalid-capability")},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							"invalid-capability",
						},
					},
				},
			})
	})
}

func TestMissingActiveGateMemoryLimit(t *testing.T) {
	t.Run("memory warning in activeGate mode", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.RoutingCapability.DisplayName,
						},
						CapabilityProperties: activegate.CapabilityProperties{
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			})
	})
	t.Run("no memory warning in activeGate mode with memory limit", func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.RoutingCapability.DisplayName,
						},
						CapabilityProperties: activegate.CapabilityProperties{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceLimitsMemory: *resource.NewMilliQuantity(1, ""),
								},
							},
						},
					},
				},
			})
	})
}

func TestActiveGatePVCSettings(t *testing.T) {
	t.Run("EphemeralVolume disabled and PVC specified", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						UseEphemeralVolume:  new(false),
						VolumeClaimTemplate: &corev1.PersistentVolumeClaimSpec{},
					},
				},
			})
	})
	t.Run("EphemeralVolume enabled and no PVC specified", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						UseEphemeralVolume: new(true),
					},
				},
			})
	})
	t.Run("EphemeralVolume enabled and PVC specified", func(t *testing.T) {
		assertDenied(t,
			[]string{errorActiveGateInvalidPVCConfiguration},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testAPIURL,
					Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
					ActiveGate: activegate.Spec{
						UseEphemeralVolume:  new(true),
						VolumeClaimTemplate: &corev1.PersistentVolumeClaimSpec{},
					},
				},
			})
	})
}

func TestActiveGateConflictingVolumes(t *testing.T) {
	t.Run("non-conflicting custom volumes are allowed", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						Volumes: []corev1.Volume{
							{Name: "my-custom-volume", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "my-custom-volume", MountPath: "/my/custom/path"},
						},
					},
				},
			},
		})
	})

	t.Run("conflict with managed volume name is denied", func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorActiveGateConflictingVolumeName, agconsts.AuthTokenSecretVolumeName)},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						CapabilityProperties: activegate.CapabilityProperties{
							Volumes: []corev1.Volume{
								{Name: agconsts.AuthTokenSecretVolumeName},
							},
						},
					},
				},
			})
	})

	t.Run("conflict with managed volume mount path is denied", func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorActiveGateConflictingVolumeMountPath, agconsts.GatewayConfigMountPath)},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						CapabilityProperties: activegate.CapabilityProperties{
							VolumeMounts: []corev1.VolumeMount{
								{MountPath: agconsts.GatewayConfigMountPath},
							},
						},
					},
				},
			})
	})

	t.Run("subpath of managed volume mount path is denied", func(t *testing.T) {
		conflictingPath := agconsts.GatewayConfigMountPath + "/subdir"
		assertDenied(t,
			[]string{fmt.Sprintf(errorActiveGateConflictingVolumeMountPath, conflictingPath)},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						CapabilityProperties: activegate.CapabilityProperties{
							VolumeMounts: []corev1.VolumeMount{
								{MountPath: conflictingPath},
							},
						},
					},
				},
			})
	})
}

func TestValidateVolumes(t *testing.T) {
	validVolume := corev1.Volume{Name: "my-vol", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
	validMount := corev1.VolumeMount{Name: "my-vol", MountPath: "/my/path"}

	t.Run("valid volume and mount", func(t *testing.T) {
		err := validateVolumes([]corev1.Volume{validVolume}, []corev1.VolumeMount{validMount})
		require.NoError(t, err)
	})

	t.Run("no volumes and no mounts", func(t *testing.T) {
		err := validateVolumes(nil, nil)
		require.NoError(t, err)
	})

	t.Run("duplicate volume name", func(t *testing.T) {
		err := validateVolumes(
			[]corev1.Volume{validVolume, validVolume},
			nil,
		)
		require.Error(t, err)
	})

	t.Run("mount without matching volume", func(t *testing.T) {
		err := validateVolumes(nil, []corev1.VolumeMount{validMount})
		require.Error(t, err)
	})

	t.Run("duplicate mount path", func(t *testing.T) {
		secondVolume := corev1.Volume{Name: "other-vol", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
		err := validateVolumes(
			[]corev1.Volume{validVolume, secondVolume},
			[]corev1.VolumeMount{
				validMount,
				{Name: "other-vol", MountPath: "/my/path"},
			},
		)
		require.Error(t, err)
	})
}

func TestActiveGateHasInvalidVolumes(t *testing.T) {
	validVolume := corev1.Volume{Name: "my-vol", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
	validMount := corev1.VolumeMount{Name: "my-vol", MountPath: "/my/path"}

	t.Run("valid volume with matching mount is allowed", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						Volumes:      []corev1.Volume{validVolume},
						VolumeMounts: []corev1.VolumeMount{validMount},
					},
				},
			},
		})
	})

	t.Run("mount without matching volume is denied", func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorActiveGateInvalidVolumes, "has volume mount without matching volume defined: my-vol")},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						CapabilityProperties: activegate.CapabilityProperties{
							VolumeMounts: []corev1.VolumeMount{validMount},
						},
					},
				},
			})
	})

	t.Run("duplicate volume name is denied", func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorActiveGateInvalidVolumes, "duplicate volume name: my-vol")},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						CapabilityProperties: activegate.CapabilityProperties{
							Volumes: []corev1.Volume{validVolume, validVolume},
						},
					},
				},
			})
	})

	t.Run("duplicate mount path is denied", func(t *testing.T) {
		secondVolume := corev1.Volume{Name: "other-vol", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
		assertDenied(t,
			[]string{fmt.Sprintf(errorActiveGateInvalidVolumes, "duplicate volume mount path: /my/path")},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						CapabilityProperties: activegate.CapabilityProperties{
							Volumes: []corev1.Volume{validVolume, secondVolume},
							VolumeMounts: []corev1.VolumeMount{
								validMount,
								{Name: "other-vol", MountPath: "/my/path"},
							},
						},
					},
				},
			})
	})
}

func TestActiveGateRollingUpdateWithGivenK8sVersion(t *testing.T) {
	maxUnavailable := intstr.FromString("20%")
	rollingUpdate := &appsv1.RollingUpdateStatefulSetStrategy{MaxUnavailable: &maxUnavailable}

	withRollingUpdate := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						RollingUpdate: rollingUpdate,
					},
				},
			},
		}
	}

	t.Run("no warning when rollingUpdate is not set", func(t *testing.T) {
		t.Cleanup(k8sversion.DisableCacheForTest(34))
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL},
		}
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("warning when rollingUpdate is set and k8s version is 1.34", func(t *testing.T) {
		t.Cleanup(k8sversion.DisableCacheForTest(34))
		assertAllowedWithWarnings(t, 1, withRollingUpdate())
	})

	t.Run("no warning when rollingUpdate is set and k8s version is 1.35", func(t *testing.T) {
		t.Cleanup(k8sversion.DisableCacheForTest(35))
		assertAllowedWithoutWarnings(t, withRollingUpdate())
	})
}
