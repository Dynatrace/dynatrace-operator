package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/spf13/afero"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestActiveGatePhaseChanges(t *testing.T) {
	mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{
		dtclient.TokenScopeDataExport,
		dtclient.TokenScopeInstallerDownload,
		dtclient.TokenScopeActiveGateTokenCreate})

	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
		},
		Status: *getTestDynkubeStatus(),
	}
	data := map[string][]byte{
		dtclient.DynatraceApiToken: []byte(testAPIToken),
	}

	objects := []client.Object{
		instance,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: data},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUID,
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.OneAgentDaemonsetName(),
				Namespace: testNamespace,
			},
		},
	}

	objects = append(objects, createTenantSecrets(instance)...)

	fakeClient := fake.NewClient(objects...)

	mockDtcBuilder := &dynatraceclient.StubBuilder{
		DynatraceClient: mockClient,
	}

	controller := &Controller{
		client:                              fakeClient,
		apiReader:                           fakeClient,
		scheme:                              scheme.Scheme,
		dynatraceClientBuilder:              mockDtcBuilder,
		fs:                                  afero.Afero{Fs: afero.NewMemMapFs()},
		registryClientBuilder:               createFakeRegistryClientBuilder(),
		deploymentMetadataReconcilerBuilder: createFakeDeploymentMetadataReconcilerBuild(),
	}
	controller.determineDynaKubePhase(instance)
}
