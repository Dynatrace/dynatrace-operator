package kubesystem

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetUID(t *testing.T) {
	const testUID = types.UID("test-uid")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: Namespace,
					UID:  testUID,
				},
			},
		).Build()
	uid, err := GetUID(fakeClient)

	assert.NoError(t, err)
	assert.NotEmpty(t, uid)
	assert.Equal(t, testUID, uid)
}

func TestDeployedViaOLM(t *testing.T) {
	testPodName := "test-pod"
	testNamespaceName := "test-namespace"

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: Namespace,
				},
			},
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testPodName,
					Namespace: testNamespaceName,
					Annotations: map[string]string{
						"olm.operatorNamespace": "operators",
					},
				},
			},
		).Build()

	_ = os.Setenv(EnvPodName, testPodName)
	_ = os.Setenv(EnvPodNamespace, testNamespaceName)

	deployed, err := DeployedViaOLM(fakeClient)

	assert.NoError(t, err)
	assert.True(t, deployed)
}
