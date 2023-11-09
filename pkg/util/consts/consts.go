package consts

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentName = "deployment-as-owner-of-secret"

	TestValue1 = "test-value"
	TestValue2 = "test-alternative-value"
	TestKey1   = "test-key"
	TestKey2   = "test-name"

	TestNamespace = "test-namespace"

	TestAppName          = "dynatrace-operator"
	TestAppVersion       = "snapshot"
	TestName             = "test-name"
	TestComponent        = "test-component"
	TestComponentFeature = "test-component-feature"
	TestComponentVersion = "test-component-version"
	UnprivilegedUser     = int64(1000)
	UnprivilegedGroup    = int64(1000)
)

var DaemonSetLog = logger.Factory.GetLogger("test-daemonset")

var ConfigMapLog = logger.Factory.GetLogger("test-configMap")

func CreateDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: DeploymentName,
		},
	}
}

func CreateTestDeploymentWithMatchLabels(name, namespace string, annotations, matchLabels map[string]string) appsv1.Deployment {
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
