package server

import (
	"encoding/json"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/extensions/metrics/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	tenantUrlScheme     = "https://dry.dev.dynatracelabs.com"
	tenantApiUrl        = tenantUrlScheme + "/api"
	tenantUuid          = "e3a4f29c343"
	tenantDynaMetricUrl = tenantUrlScheme + "/e/" + tenantUuid

	dynaMetricImage = "nowhere/synthetic-adapter-amd64:archaic"

	dynaMetricCertificateDirArg = "--cert-dir=" + tmpStorageMountPath
)

var (
	dynaMetricPortArg = "--secure-port=" + fmt.Sprint(common.HttpsServicePort)

	dynakube = &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ephemeral",
			Namespace: "experimental",
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId: "doctored",
				dynatracev1beta1.AnnotationFeatureCustomDynaMetricImage:     dynaMetricImage,
			},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: tenantApiUrl,
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			ConnectionInfo: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: tenantUuid,
			},
		},
	}
)

func TestDynaMetricDeployment(t *testing.T) {
	assertion := assert.New(t)
	deployment, err := newBuilder(dynakube).newDeployment()

	toAssertImage := func(t *testing.T) {
		assertion.NoError(err)
		assertion.Equal(
			deployment.Spec.Template.Spec.Containers[0].Image,
			dynaMetricImage,
			"declared custom image: %s",
			dynaMetricImage)
	}
	t.Run("by-image", toAssertImage)

	toAssertEnvironement := func(t *testing.T) {
		envBindings := []corev1.EnvVar{
			{
				Name:  envBaseUrl,
				Value: tenantDynaMetricUrl,
			},
		}
		assertion.Subset(
			deployment.Spec.Template.Spec.Containers[0].Env,
			envBindings,
			"declared env variables")
		jsonized, _ := json.Marshal(deployment)
		t.Logf("manifest:\n%s", jsonized)
	}
	t.Run("by-env-variables", toAssertEnvironement)

	toAssertArguments := func(t *testing.T) {
		args := []string{
			dynaMetricPortArg,
			dynaMetricCertificateDirArg,
		}
		assertion.ElementsMatch(
			deployment.Spec.Template.Spec.Containers[0].Args,
			args,
			"declared cmd args")
	}
	t.Run("by-command-arguments", toAssertArguments)
}
