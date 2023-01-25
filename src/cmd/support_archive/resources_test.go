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
	clientmocks "github.com/Dynatrace/dynatrace-operator/src/mocks/sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testOperatorNamespace = "dynatrace"

//go:generate mockery --case=snake --srcpkg=sigs.k8s.io/controller-runtime/pkg/client --name=Reader --output ../../mocks/sigs.k8s.io/controller-runtime/pkg/client

func TestManifestCollector(t *testing.T) {
	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
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
				TypeMeta: typeMeta("Namespace"),
				ObjectMeta: metav1.ObjectMeta{
					Name: "some-app-namespace",
					Labels: map[string]string{
						webhook.InjectionInstanceLabel: "abc12345",
					},
				},
			},
			&corev1.Namespace{
				TypeMeta: typeMeta("Namespace"),
				ObjectMeta: metav1.ObjectMeta{
					Name: testOperatorNamespace,
				},
			},
			&corev1.Namespace{
				TypeMeta: typeMeta("Namespace"),
				ObjectMeta: metav1.ObjectMeta{
					Name: "uninjectednamespace",
				},
			},
			&dynatracev1beta1.DynaKube{
				TypeMeta:   typeMeta("DynaKube"),
				ObjectMeta: objectMeta("dynakube1"),
			},
		).Build()

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	ctx := context.TODO()
	mockedApiReader := clientmocks.NewReader(t)
	queries := getQueries(testOperatorNamespace)

	for _, q := range queries {
		mockedApiReader.
			On("List", toFlatInterfaceSlice(ctx, getExpectedObjectsList(q.groupVersionKind), q.filters)...).
			Return(clt.List)
	}

	require.NoError(t, newK8sObjectCollector(ctx, log, supportArchive, testOperatorNamespace, mockedApiReader).Do())

	expectedFiles := []string{
		fmt.Sprintf("%s/Namespace-some-app-namespace.json", InjectedNamespacesManifestsDirectoryName),
		fmt.Sprintf("%s/Namespace-dynatrace.json", testOperatorNamespace),

		// fake.Client does not respect the field selector of the list options in query_objects.go
		// therefore the resourceQuery for the operator namespace returns all namespaces configured above
		// That's why these two namespaces are expected in this test, although they should actually be filtered out.
		// This sucks, but it works in production code because client.Client.List respects the field selector filter.
		fmt.Sprintf("%s/Namespace-some-app-namespace.json", InjectedNamespacesManifestsDirectoryName),
		fmt.Sprintf("%s/Namespace-uninjectednamespace.json", InjectedNamespacesManifestsDirectoryName),

		fmt.Sprintf("%s/Deployment/deployment1.json", testOperatorNamespace),
		fmt.Sprintf("%s/StatefulSet/statefulset1.json", testOperatorNamespace),
		fmt.Sprintf("%s/DaemonSet/daemonset1.json", testOperatorNamespace),
		fmt.Sprintf("%s/DynaKube/dynakube1.json", testOperatorNamespace),
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

func TestManifestCollectorNoManifestsAvailable(t *testing.T) {
	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	ctx := context.TODO()
	mockedApiReader := clientmocks.NewReader(t)
	queries := getQueries(testOperatorNamespace)

	for _, q := range queries {
		mockedApiReader.
			On("List", toFlatInterfaceSlice(ctx, getExpectedObjectsList(q.groupVersionKind), q.filters)...).
			Return(clt.List)
	}

	require.NoError(t, newK8sObjectCollector(ctx, log, supportArchive, testOperatorNamespace, mockedApiReader).Do())

	tarReader := tar.NewReader(&tarBuffer)
	_, err := tarReader.Next()
	require.ErrorIs(t, err, io.EOF)
}

func TestManifestCollectionFails(t *testing.T) {
	mockedApiReader := clientmocks.NewReader(t)

	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&appsv1.Deployment{
				TypeMeta:   typeMeta("Deployment"),
				ObjectMeta: objectMeta("deployment1"),
			},
			&appsv1.StatefulSet{
				TypeMeta:   typeMeta("StatefulSet"),
				ObjectMeta: objectMeta("statefulset1"),
			},
			&corev1.Namespace{
				TypeMeta: typeMeta("Namespace"),
				ObjectMeta: metav1.ObjectMeta{
					Name:         "some-app-namespace",
					GenerateName: "",
					Labels: map[string]string{
						webhook.InjectionInstanceLabel: "abc12345",
					},
				},
			},
			&corev1.Namespace{
				TypeMeta: typeMeta("Namespace"),
				ObjectMeta: metav1.ObjectMeta{
					Name:         testOperatorNamespace,
					GenerateName: "",
				},
			},
			&dynatracev1beta1.DynaKube{
				TypeMeta:   typeMeta("DynaKube"),
				ObjectMeta: objectMeta("dynakube1"),
			},
		).Build()

	context := context.TODO()

	queries := getQueries(testOperatorNamespace)
	require.Len(t, queries, 9)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[0].groupVersionKind), queries[0].filters)...).
		Return(clt.List)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[1].groupVersionKind), queries[1].filters)...).
		// make sure an error doesn't stop further processing
		Return(assert.AnError)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[2].groupVersionKind), queries[2].filters)...).
		Return(assert.AnError)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[3].groupVersionKind), queries[3].filters)...).
		// here an empty DaemonSet list will be produced, make sure it doesn't stop further processing
		Return(clt.List)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[4].groupVersionKind), queries[4].filters)...).
		Return(clt.List)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[5].groupVersionKind), queries[5].filters)...).
		Return(clt.List)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[6].groupVersionKind), queries[6].filters)...).
		Return(clt.List)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[7].groupVersionKind), queries[7].filters)...).
		Return(clt.List)

	mockedApiReader.
		On("List", toFlatInterfaceSlice(context, getExpectedObjectsList(queries[8].groupVersionKind), queries[8].filters)...).
		Return(clt.List)

	tarBuffer := bytes.Buffer{}
	supportArchive := tarball{
		tarWriter: tar.NewWriter(&tarBuffer),
	}

	collector := newK8sObjectCollector(context, log, supportArchive, testOperatorNamespace, mockedApiReader)
	require.NoError(t, collector.Do())

	tarReader := tar.NewReader(&tarBuffer)

	hdr, err := tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, expectedFilename("injected_namespaces/Namespace-some-app-namespace.json"), hdr.Name)

	hdr, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/StatefulSet/statefulset1.json", testOperatorNamespace)), hdr.Name)

	hdr, err = tarReader.Next()
	require.NoError(t, err)
	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/DynaKube/dynakube1.json", testOperatorNamespace)), hdr.Name)
}

func getExpectedObjectsList(groupVersionKind schema.GroupVersionKind) *unstructured.UnstructuredList {
	expectedObjectsQuery := &unstructured.UnstructuredList{}
	expectedObjectsQuery.SetGroupVersionKind(groupVersionKind)
	return expectedObjectsQuery
}

func toInterfaceSlice(slice []client.ListOption) []interface{} {
	ifSlice := make([]interface{}, 0)
	for _, elem := range slice {
		ifSlice = append(ifSlice, elem)
	}
	return ifSlice
}

func toFlatInterfaceSlice(args ...interface{}) []interface{} {
	ifSlice := make([]interface{}, 0)
	for _, arg := range args {
		switch elem := arg.(type) {
		case []client.ListOption:
			ifSlice = append(ifSlice, toInterfaceSlice(elem)...)
		default:
			ifSlice = append(ifSlice, elem)
		}
	}
	return ifSlice
}

func typeMeta(kind string) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       kind,
		APIVersion: "v1",
	}
}

func objectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:         name,
		GenerateName: "",
		Namespace:    testOperatorNamespace,
		Labels: map[string]string{
			kubeobjects.AppNameLabel: "dynatrace-operator",
		},
	}
}

func expectedFilename(objname string) string {
	return fmt.Sprintf("%s/%s", ManifestsDirectoryName, objname)
}
