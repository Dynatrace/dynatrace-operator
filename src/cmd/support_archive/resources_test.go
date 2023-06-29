package support_archive

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testOperatorNamespace = "dynatrace"
const manifestExtension = ".yaml"

func TestManifestCollector_Success(t *testing.T) {
	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	clt := fake.NewClientWithIndex(
		&appsv1.Deployment{
			TypeMeta:   typeMeta("Deployment"),
			ObjectMeta: objectMeta("deployment1"),
		},
		&appsv1.DaemonSet{
			TypeMeta:   typeMeta("DaemonSet"),
			ObjectMeta: objectMeta("daemonset1"),
		},
		&appsv1.StatefulSet{
			TypeMeta:   typeMeta("StatefulSet"),
			ObjectMeta: objectMeta("statefulset1"),
		},
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "corev1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "uninjectednamespace",
				Labels: map[string]string{
					"random": "label",
				},
			},
		},
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core/v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "some-app-namespace",
				Labels: map[string]string{
					webhook.InjectionInstanceLabel: "abc12345",
				},
			},
		},
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core/v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "random",
			},
		},
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core/v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: testOperatorNamespace,
			},
		},
		&dynatracev1beta1.DynaKube{
			TypeMeta:   typeMeta("DynaKube"),
			ObjectMeta: objectMeta("dynakube1"),
		},
	)

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	ctx := context.TODO()
	require.NoError(t, newK8sObjectCollector(ctx, log, supportArchive, testOperatorNamespace, defaultOperatorAppName, clt).Do())

	expectedFiles := []string{
		fmt.Sprintf("%s/namespace-some-app-namespace%s", InjectedNamespacesManifestsDirectoryName, manifestExtension),
		fmt.Sprintf("%s/namespace-dynatrace%s", testOperatorNamespace, manifestExtension),

		fmt.Sprintf("%s/deployment/deployment1%s", testOperatorNamespace, manifestExtension),
		fmt.Sprintf("%s/statefulset/statefulset1%s", testOperatorNamespace, manifestExtension),
		fmt.Sprintf("%s/daemonset/daemonset1%s", testOperatorNamespace, manifestExtension),
		fmt.Sprintf("%s/dynakube/dynakube1%s", testOperatorNamespace, manifestExtension),
	}

	tarReader := tar.NewReader(&tarBuffer)

	for _, expectedFile := range expectedFiles {
		t.Run("expected "+expectedFile, func(t *testing.T) {
			hdr, err := tarReader.Next()
			require.NoError(t, err)
			assert.Equal(t, expectedFilename(expectedFile), hdr.Name)
		})
	}

	_, err := tarReader.Next()
	require.ErrorIs(t, err, io.EOF)
}

func TestManifestCollector_NoManifestsAvailable(t *testing.T) {
	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	clt := fake.NewClientWithIndex()

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	ctx := context.TODO()

	err := newK8sObjectCollector(ctx, log, supportArchive, testOperatorNamespace, defaultOperatorAppName, clt).Do()
	require.NoError(t, err)

	tarReader := tar.NewReader(&tarBuffer)
	_, err = tarReader.Next()
	require.ErrorIs(t, err, io.EOF)
}

func TestManifestCollector_PartialCollectionOnMissingResources(t *testing.T) {
	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	queries := getQueries(testOperatorNamespace, defaultOperatorAppName)
	require.Len(t, queries, 9)

	clt := fake.NewClientWithIndex(
		&appsv1.StatefulSet{
			TypeMeta:   typeMeta("StatefulSet"),
			ObjectMeta: objectMeta("statefulset1"),
		},
		&corev1.Namespace{
			TypeMeta: typeMeta("Namespace"),
			ObjectMeta: metav1.ObjectMeta{
				Name: "some-app-namespace",
				Labels: map[string]string{
					webhook.InjectionInstanceLabel: "abc12345",
				},
			},
		},
		&dynatracev1beta1.DynaKube{
			TypeMeta:   typeMeta("DynaKube"),
			ObjectMeta: objectMeta("dynakube1"),
		},
	)

	context := context.TODO()

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	collector := newK8sObjectCollector(context, log, supportArchive, testOperatorNamespace, defaultOperatorAppName, clt)
	require.NoError(t, collector.Do())

	tarReader := tar.NewReader(&tarBuffer)

	hdr, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, expectedFilename(fmt.Sprintf("injected_namespaces/namespace-some-app-namespace%s", manifestExtension)), hdr.Name)

	hdr, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/statefulset/statefulset1%s", testOperatorNamespace, manifestExtension)), hdr.Name)

	hdr, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/dynakube/dynakube1%s", testOperatorNamespace, manifestExtension)), hdr.Name)

	_, err = tarReader.Next()
	require.ErrorIs(t, err, io.EOF)
}

func typeMeta(kind string) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       kind,
		APIVersion: "v1",
	}
}

func objectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: testOperatorNamespace,
		Labels: map[string]string{
			kubeobjects.AppNameLabel: "dynatrace-operator",
		},
	}
}

func expectedFilename(objname string) string {
	return fmt.Sprintf("%s/%s", ManifestsDirectoryName, objname)
}
