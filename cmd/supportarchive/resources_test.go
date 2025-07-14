package supportarchive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
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
		&dynakube.DynaKube{
			TypeMeta:   typeMeta("DynaKube"),
			ObjectMeta: objectMeta("dynakube1"),
		},
		&edgeconnect.EdgeConnect{
			TypeMeta:   typeMeta("EdgeConnect"),
			ObjectMeta: objectMeta("edgeconnect1"),
		},
		&admissionregistrationv1.MutatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "admissionregistration.k8s.io/v1",
				Kind:       "MutatingWebhookConfiguration",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynatrace-webhook",
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "admissionregistration.k8s.io/v1",
				Kind:       "ValidatingWebhookConfiguration",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynatrace-webhook",
			},
		},
		&v1.CustomResourceDefinition{
			TypeMeta: typeMeta("CustomResourceDefinition"),
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakubes.dynatrace.com",
			},
		},
		&v1.CustomResourceDefinition{
			TypeMeta: typeMeta("CustomResourceDefinition"),
			ObjectMeta: metav1.ObjectMeta{
				Name: "edgeconnects.dynatrace.com",
			},
		},
	)

	buffer := bytes.Buffer{}
	supportArchive := newZipArchive(bufio.NewWriter(&buffer))

	client := fakeclientset.NewSimpleClientset()

	client.Resources = getResourceLists()
	fakeDiscovery, _ := client.Discovery().(*fakediscovery.FakeDiscovery)

	require.NoError(t, newK8sObjectCollector(context.Background(), log, supportArchive, testOperatorNamespace, defaultOperatorAppName, clt, fakeDiscovery).Do())
	assertNoErrorOnClose(t, supportArchive)

	expectedFiles := []string{
		fmt.Sprintf("%s/namespace-some-app-namespace%s", InjectedNamespacesManifestsDirectoryName, manifestExtension),
		fmt.Sprintf("%s/namespace-dynatrace%s", testOperatorNamespace, manifestExtension),

		fmt.Sprintf("%s/deployment/deployment1%s", testOperatorNamespace, manifestExtension),
		fmt.Sprintf("%s/statefulset/statefulset1%s", testOperatorNamespace, manifestExtension),
		fmt.Sprintf("%s/daemonset/daemonset1%s", testOperatorNamespace, manifestExtension),
		fmt.Sprintf("%s/dynakube/dynakube1%s", testOperatorNamespace, manifestExtension),
		fmt.Sprintf("%s/edgeconnect/edgeconnect1%s", testOperatorNamespace, manifestExtension),
		fmt.Sprintf("%s/mutatingwebhookconfiguration%s", "webhook_configurations", manifestExtension),
		fmt.Sprintf("%s/validatingwebhookconfiguration%s", "webhook_configurations", manifestExtension),
		fmt.Sprintf("%s/customresourcedefinition-dynakubes%s", "crds", manifestExtension),
		fmt.Sprintf("%s/customresourcedefinition-edgeconnects%s", "crds", manifestExtension),
	}

	sort.Strings(expectedFiles)

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))

	actualFileName := make([]string, len(zipReader.File))

	for i, file := range zipReader.File {
		actualFileName[i] = file.Name
	}

	sort.Strings(actualFileName)

	for i, expectedFile := range expectedFiles {
		t.Run("expected "+expectedFile, func(t *testing.T) {
			require.NoError(t, err)
			assert.Equal(t, expectedFilename(expectedFile), actualFileName[i])
		})
	}
}

func TestManifestCollector_NoManifestsAvailable(t *testing.T) {
	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	clt := fake.NewClientWithIndex()

	buffer := bytes.Buffer{}
	supportArchive := newZipArchive(bufio.NewWriter(&buffer))

	client := fakeclientset.NewSimpleClientset()
	fakeDiscovery, _ := client.Discovery().(*fakediscovery.FakeDiscovery)

	err := newK8sObjectCollector(context.Background(), log, supportArchive, testOperatorNamespace, defaultOperatorAppName, clt, fakeDiscovery).Do()
	require.NoError(t, err)
	assertNoErrorOnClose(t, supportArchive)

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	assert.Empty(t, zipReader.File)
}

func TestManifestCollector_PartialCollectionOnMissingResources(t *testing.T) {
	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	queries := getQueries(testOperatorNamespace, defaultOperatorAppName)
	require.Len(t, queries, 18)

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
		&dynakube.DynaKube{
			TypeMeta:   typeMeta("DynaKube"),
			ObjectMeta: objectMeta("dynakube1"),
		},
		&edgeconnect.EdgeConnect{
			TypeMeta:   typeMeta("EdgeConnect"),
			ObjectMeta: objectMeta("edgeconnect1"),
		},
		&admissionregistrationv1.MutatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "admissionregistration.k8s.io/v1",
				Kind:       "MutatingWebhookConfiguration",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynatrace-webhook",
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "admissionregistration.k8s.io/v1",
				Kind:       "ValidatingWebhookConfiguration",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynatrace-webhook",
			},
		},
		&v1.CustomResourceDefinition{
			TypeMeta: typeMeta("CustomResourceDefinition"),
			ObjectMeta: metav1.ObjectMeta{
				Name: "dynakubes.dynatrace.com",
			},
		},
		&v1.CustomResourceDefinition{
			TypeMeta: typeMeta("CustomResourceDefinition"),
			ObjectMeta: metav1.ObjectMeta{
				Name: "edgeconnects.dynatrace.com",
			},
		},
	)

	buffer := bytes.Buffer{}
	supportArchive := newZipArchive(bufio.NewWriter(&buffer))

	client := fakeclientset.NewSimpleClientset()

	client.Resources = getResourceLists()

	fakeDiscovery := client.Discovery().(*fakediscovery.FakeDiscovery)

	collector := newK8sObjectCollector(context.Background(), log, supportArchive, testOperatorNamespace, defaultOperatorAppName, clt, fakeDiscovery)
	require.NoError(t, collector.Do())
	assertNoErrorOnClose(t, supportArchive)

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	require.Len(t, zipReader.File, 8)
	assert.Equal(t, expectedFilename(fmt.Sprintf("injected_namespaces/namespace-some-app-namespace%s", manifestExtension)), zipReader.File[0].Name)

	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/statefulset/statefulset1%s", testOperatorNamespace, manifestExtension)), zipReader.File[1].Name)

	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/dynakube/dynakube1%s", testOperatorNamespace, manifestExtension)), zipReader.File[2].Name)

	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/edgeconnect/edgeconnect1%s", testOperatorNamespace, manifestExtension)), zipReader.File[3].Name)

	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/mutatingwebhookconfiguration%s", "webhook_configurations", manifestExtension)), zipReader.File[4].Name)

	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/validatingwebhookconfiguration%s", "webhook_configurations", manifestExtension)), zipReader.File[5].Name)

	crds := []string{zipReader.File[6].Name, zipReader.File[7].Name}
	sort.Strings(crds)
	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/customresourcedefinition-dynakubes%s", "crds", manifestExtension)), crds[0])

	assert.Equal(t, expectedFilename(fmt.Sprintf("%s/customresourcedefinition-edgeconnects%s", "crds", manifestExtension)), crds[1])
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
			labels.AppNameLabel: "dynatrace-operator",
		},
	}
}

func expectedFilename(objname string) string {
	return fmt.Sprintf("%s/%s", ManifestsDirectoryName, objname)
}

func getResourceLists() []*metav1.APIResourceList {
	stable := metav1.APIResourceList{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "pods", Namespaced: true, Kind: "Pod"},
			{Name: "services", Namespaced: true, Kind: "Service"},
			{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
		},
	}
	dk := metav1.APIResourceList{
		GroupVersion: crdNameSuffix + "/" + "v1beta2",
		APIResources: []metav1.APIResource{
			{Version: "v1beta2", Group: crdNameSuffix, Name: "dynakubes", Namespaced: true, Kind: "DynaKube"},
		},
	}
	ec := metav1.APIResourceList{
		GroupVersion: crdNameSuffix + "/" + "v1alpha1",
		APIResources: []metav1.APIResource{
			{Version: "v1alpha2", Group: crdNameSuffix, Name: "edgeconnects", Namespaced: true, Kind: "EdgeConnect"},
		},
	}

	return []*metav1.APIResourceList{
		&stable,
		&dk,
		&ec,
	}
}
