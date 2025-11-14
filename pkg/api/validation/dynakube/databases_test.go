package validation

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/databases"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConflictingOrInvalidVolumeMounts(t *testing.T) {
	baseDk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: &extensions.Spec{
				Databases: []extensions.DatabaseSpec{},
			},
		},
	}

	t.Run("no databases defined => no error", func(t *testing.T) {
		dk := baseDk.DeepCopy()
		response := conflictingOrInvalidDatabasesVolumeMounts(t.Context(), nil, dk)

		assert.Empty(t, response)
	})

	t.Run("non-conflicting paths => no error", func(t *testing.T) {
		volumeName := "foo"

		dbSpec := extensions.DatabaseSpec{
			ID: "some-db",
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volumeName,
					MountPath: "/bar/baz",
				},
			},
		}

		dk := baseDk.DeepCopy()
		dk.Spec.Extensions.Databases = append(dk.Spec.Extensions.Databases, dbSpec)

		response := conflictingOrInvalidDatabasesVolumeMounts(t.Context(), nil, dk)

		assert.Empty(t, response)
	})

	t.Run("default volume mount used => conflicts => fail", func(t *testing.T) {
		dbSpec := extensions.DatabaseSpec{
			ID:           testName,
			Volumes:      []corev1.Volume{},
			VolumeMounts: []corev1.VolumeMount{},
		}
		dk := baseDk.DeepCopy()

		defaultVolumeMount := databases.GetDefaultVolumeMounts(dk)[0]
		volumeName := "some-volume"
		volumeMountPath := filepath.Join(defaultVolumeMount.MountPath, "some", "sub", "path")
		dbSpec.Volumes = append(dbSpec.Volumes, corev1.Volume{Name: volumeName})
		dbSpec.VolumeMounts = append(dbSpec.VolumeMounts, corev1.VolumeMount{Name: volumeName, MountPath: volumeMountPath})

		dk.Spec.Extensions.Databases = append(dk.Spec.Extensions.Databases, dbSpec)

		expectedError := fmt.Sprintf(errorConflictingDatabasesVolumeMounts, volumeMountPath, defaultVolumeMount.MountPath)
		actualError := conflictingOrInvalidDatabasesVolumeMounts(t.Context(), nil, dk)

		assert.Equal(t, expectedError, actualError)
	})

	t.Run("invalid volume mount path => error", func(t *testing.T) {
		volumeName := "vol123"
		volumeMountPath := "not/absolute"

		dbSpec := extensions.DatabaseSpec{
			ID: testName,
			Volumes: []corev1.Volume{
				{Name: volumeName},
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: volumeName, MountPath: volumeMountPath},
			},
		}

		dk := baseDk.DeepCopy()
		dk.Spec.Extensions.Databases = append(dk.Spec.Extensions.Databases, dbSpec)
		response := conflictingOrInvalidDatabasesVolumeMounts(t.Context(), nil, dk)

		assert.Equal(t, fmt.Sprintf(errorInvalidDatabasesVolumeMountPath, volumeMountPath), response)
	})
}

func TestUnusedVolumes(t *testing.T) {
	baseDk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: &extensions.Spec{
				Databases: []extensions.DatabaseSpec{},
			},
		},
	}

	t.Run("unused volumes => error", func(t *testing.T) {
		volumeNames := []string{"definitely-unused", "also-unused"}

		dbSpec := extensions.DatabaseSpec{ID: testName, Volumes: []corev1.Volume{}}
		for _, name := range volumeNames {
			dbSpec.Volumes = append(dbSpec.Volumes, corev1.Volume{Name: name})
		}

		dk := baseDk.DeepCopy()
		dk.Spec.Extensions.Databases = append(dk.Spec.Extensions.Databases, dbSpec)

		expectedError := fmt.Sprintf(errorUnusedDatabasesVolumes, strings.Join(volumeNames, ", "))
		actualError := unusedDatabasesVolume(t.Context(), nil, dk)

		assert.Equal(t, expectedError, actualError)
	})

	t.Run("no unused volumes => no error", func(t *testing.T) {
		volumeMounts := []string{"/some/path", "/some/other/path"}

		dbSpec := extensions.DatabaseSpec{ID: testName, Volumes: []corev1.Volume{}, VolumeMounts: []corev1.VolumeMount{}}
		for index, path := range volumeMounts {
			name := fmt.Sprintf("volume-%d", index)
			dbSpec.Volumes = append(dbSpec.Volumes, corev1.Volume{Name: name})
			dbSpec.VolumeMounts = append(dbSpec.VolumeMounts, corev1.VolumeMount{Name: name, MountPath: path})
		}

		dk := baseDk.DeepCopy()
		dk.Spec.Extensions.Databases = append(dk.Spec.Extensions.Databases, dbSpec)

		response := unusedDatabasesVolume(t.Context(), nil, dk)

		assert.Empty(t, response)
	})
}

func TestHostPathDatabaseVolume(t *testing.T) {
	t.Run("hostPath volume defined => error", func(t *testing.T) {
		volumeName := "illegal-host-path-volume"

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				Extensions: &extensions.Spec{
					Databases: []extensions.DatabaseSpec{
						{
							ID: testName,
							Volumes: []corev1.Volume{
								{
									Name: volumeName,
									VolumeSource: corev1.VolumeSource{
										HostPath: &corev1.HostPathVolumeSource{
											Path: "/some/host/path",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: "/foo/bar",
								},
							},
						},
					},
				},
			},
		}

		response := hostPathDatabaseVolumeFound(t.Context(), nil, dk)

		assert.Equal(t, warningHostPathDatabaseVolumeDetected, response)
	})
}
