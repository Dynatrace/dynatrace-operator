package cluster_intel_collector

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestManifestCollector(t *testing.T) {
	const namespace = "dynatrace"
	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         "deployment1",
					GenerateName: "",
					Namespace:    namespace,
				},
			},
			&appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DaemonSet",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         "daemonset1",
					GenerateName: "",
					Namespace:    namespace,
				},
			},
			&appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         "statefulset1",
					GenerateName: "",
					Namespace:    namespace,
				},
			},
			&corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         namespace,
					GenerateName: "",
				},
			},
			&v1beta1.DynaKube{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DynaKube",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:         "dynakube1",
					GenerateName: "",
					Namespace:    namespace,
				},
			},
		).Build()

	ctx := intelCollectorContext{
		ctx:           context.TODO(),
		clientSet:     nil,
		apiReader:     clt,
		namespaceName: namespace,
		toStdout:      false,
		targetDir:     "",
	}

	tarBuffer := bytes.Buffer{}
	tarball := intelTarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	require.NoError(t, collectManifests(&ctx, &tarball))
	tarReader := tar.NewReader(&tarBuffer)

	hdr, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, "Namespace-dynatrace.yaml", hdr.Name)

	hdr, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Deployment-%s-deployment1.yaml", namespace), hdr.Name)

	hdr, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("DaemonSet-%s-daemonset1.yaml", namespace), hdr.Name)

	hdr, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("StatefulSet-%s-statefulset1.yaml", namespace), hdr.Name)

	hdr, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("DynaKube-%s-dynakube1.yaml", namespace), hdr.Name)
}

func TestManifestCollectorNoManifestsAvailable(t *testing.T) {
	const namespace = "dynatrace"
	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()

	ctx := intelCollectorContext{
		ctx:           context.TODO(),
		clientSet:     nil,
		apiReader:     clt,
		namespaceName: namespace,
		toStdout:      false,
		targetDir:     "",
	}

	tarBuffer := bytes.Buffer{}
	tarball := intelTarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	require.NoError(t, collectManifests(&ctx, &tarball))
	tarReader := tar.NewReader(&tarBuffer)

	_, err := tarReader.Next()
	require.Error(t, err)
}
