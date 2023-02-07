package service

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	dynaKube = &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ephemeral",
			Namespace: "experimental",
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId: "doctored",
			},
		},
	}

	deployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transient",
			Namespace: "experimental",
			Labels: map[string]string{
				"ignored": "undefined",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"supplementary": "undefined",
				},
			},
		},
	}
)

func TestDynaMetricsService(t *testing.T) {
	assertion := assert.New(t)
	service := newBuilder(dynaKube, deployment).newService()

	toAssertLabels := func(t *testing.T) {
		assertion.Equal(
			service.ObjectMeta.Labels,
			deployment.ObjectMeta.Labels,
			"declared labels: %v",
			deployment.ObjectMeta.Labels)
	}
	t.Run("by-labels", toAssertLabels)

	toAssertMatchLabels := func(t *testing.T) {
		assertion.Equal(
			service.Spec.Selector,
			deployment.Spec.Selector.MatchLabels,
			"declared match labels: %v",
			deployment.Spec.Selector.MatchLabels)
	}
	t.Run("by-match-labels", toAssertMatchLabels)
}

func TestDynaMetricsApiService(t *testing.T) {
	assertion := assert.New(t)
	apiService := newBuilder(dynaKube, deployment).newApiService()

	serviceId := []string{
		deployment.Name,
		deployment.Namespace,
	}
	toAssertServiceIdentity := func(t *testing.T) {
		assertion.Equal(
			[]string{
				apiService.Spec.Service.Name,
				apiService.Spec.Service.Namespace,
			},
			serviceId,
			"declared service identity: %v",
			serviceId)
	}
	t.Run("by-service-identity", toAssertServiceIdentity)
}
