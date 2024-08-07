package deployment

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var deploymentLog = logd.Get().WithName("test-deployment")

func createTestDeploymentWithMatchLabels(name, namespace string, annotations, matchLabels map[string]string) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}

// GetDeployment returns the Deployment object who is the owner of this pod.
func TestGetDeployment(t *testing.T) {
	const ns = "dynatrace"

	trueVar := true

	fakeClient := fake.NewClient(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mypod",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet", Name: "myreplicaset", Controller: &trueVar},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myreplicaset",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "mydeployment", Controller: &trueVar},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydeployment",
				Namespace: ns,
			},
		})

	deploy, err := GetDeployment(fakeClient, "mypod", "dynatrace")
	require.NoError(t, err)
	assert.Equal(t, "mydeployment", deploy.Name)
	assert.Equal(t, "dynatrace", deploy.Namespace)
}
