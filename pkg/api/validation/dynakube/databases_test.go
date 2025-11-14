package validation

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/databases"
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
			APIURL: testAPIURL,
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: extensions.ExecutionControllerSpec{
					ImageRef: image.Ref{
						Repository: "some-repo",
						Tag:        "some-tag",
					},
				},
			},
			Extensions: &extensions.Spec{
				Databases: []extensions.DatabaseSpec{},
			},
		},
	}

	t.Run("no databases defined => no error", func(t *testing.T) {
		dk := baseDk.DeepCopy()

		assertAllowedWithoutWarnings(t, dk)
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

		assertAllowedWithWarnings(t, 2, dk)
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

		expectedErrors := []string{
			fmt.Sprintf(errorConflictingDatabasesVolumeMounts, volumeMountPath, defaultVolumeMount.MountPath),
		}
		assertDenied(t, expectedErrors, dk)
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

		expectedErrors := []string{
			fmt.Sprintf(errorInvalidDatabasesVolumeMountPath, volumeMountPath),
		}
		assertDenied(t, expectedErrors, dk)
	})
}

func TestUnusedVolumes(t *testing.T) {
	baseDk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: extensions.ExecutionControllerSpec{
					ImageRef: image.Ref{
						Repository: "some-repo",
						Tag:        "some-tag",
					},
				},
			},
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

		expectedErrors := []string{
			fmt.Sprintf(errorUnusedDatabasesVolumes, strings.Join(volumeNames, ", ")),
		}
		assertDenied(t, expectedErrors, dk)
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

		assertAllowedWithWarnings(t, 2, dk)
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
				APIURL: testAPIURL,
				Templates: dynakube.TemplatesSpec{
					ExtensionExecutionController: extensions.ExecutionControllerSpec{
						ImageRef: image.Ref{
							Repository: "some-repo",
							Tag:        "some-tag",
						},
					},
				},
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

		assertAllowedWithWarnings(t, 3, dk)
	})
}
