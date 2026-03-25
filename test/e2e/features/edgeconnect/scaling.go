//go:build e2e

package edgeconnect

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	ecComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8shpa"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	scaleReplicas = ptr.To(int32(3))
	baseReplicas  = ptr.To(int32(2))
)

func WithHPARegular(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-regualar-with-hpa-regular")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)

	builder.Assess("create EC configuration on the tenant", ecComponents.CreateTenantConfig(testECname, secretConfig, edgeConnectTenantConfig, testHostPattern))

	testEdgeConnect := *ecComponents.New(
		// this tenantConfigName should match with tenant edgeConnect tenantConfigName
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", secretConfig.TenantUID)),
	)

	// create OAuth client secret related to the specific EdgeConnect configuration on the tenant
	builder.Assess("create client secret", tenant.CreateClientSecret(&edgeConnectTenantConfig.Secret, ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	ecComponents.Install(builder, helpers.LevelAssess, nil, testEdgeConnect)

	testHPA := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-autoscaler",
			Namespace: testEdgeConnect.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       testECname,
				APIVersion: "apps/v1",
			},
			MinReplicas: scaleReplicas,
			MaxReplicas: *scaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if the EC deployment has replicas autoscaled to 3", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *scaleReplicas))

	builder.Teardown(k8shpa.Delete(testHPA))
	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(ecComponents.DeleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}

func WithHPAProvisioner(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-with-hpa-provisioner")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)

	testEdgeConnect := *ecComponents.New(
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(secretConfig.Resource),
		ecComponents.WithProvisionerMode(true),
		ecComponents.WithHostPattern(testHostPattern),
	)

	ecComponents.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))

	testHPA := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-autoscaler",
			Namespace: testEdgeConnect.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       testECname,
				APIVersion: "apps/v1",
			},
			MinReplicas: scaleReplicas,
			MaxReplicas: *scaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if the EC deployment has replicas autoscaled to 3", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *scaleReplicas))

	builder.Teardown(k8shpa.Delete(testHPA))
	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(ecComponents.DeleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}

func EnforceReplicasRegular(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-enforce-replicas-regular")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)

	builder.Assess("create EC configuration on the tenant", ecComponents.CreateTenantConfig(testECname, secretConfig, edgeConnectTenantConfig, testHostPattern))

	testEdgeConnect := *ecComponents.New(
		// this tenantConfigName should match with tenant edgeConnect tenantConfigName
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", secretConfig.TenantUID)),
		ecComponents.WithReplicas(baseReplicas),
	)

	// create OAuth client secret related to the specific EdgeConnect configuration on the tenant
	builder.Assess("create client secret", tenant.CreateClientSecret(&edgeConnectTenantConfig.Secret, ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	ecComponents.Install(builder, helpers.LevelAssess, nil, testEdgeConnect)

	builder.Assess("scale EC deployment replicas to 3", k8sdeployment.Update(testEdgeConnect.Name, testEdgeConnect.Namespace, func(d *appsv1.Deployment) {
		d.Spec.Replicas = scaleReplicas
	}))

	builder.Assess("check if the EC deployment has roll back replicas set to 2", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *baseReplicas))

	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(ecComponents.DeleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}

func EnforceReplicasProvisioner(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-enforce-replicas-provisioner")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)

	testEdgeConnect := *ecComponents.New(
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(secretConfig.Resource),
		ecComponents.WithProvisionerMode(true),
		ecComponents.WithHostPattern(testHostPattern),
		ecComponents.WithReplicas(baseReplicas),
	)

	ecComponents.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))

	builder.Assess("scale EC deployment replicas to 3", k8sdeployment.Update(testEdgeConnect.Name, testEdgeConnect.Namespace, func(d *appsv1.Deployment) {
		d.Spec.Replicas = scaleReplicas
	}))

	builder.Assess("check if the EC deployment has roll back replicas set to 2", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *baseReplicas))

	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(ecComponents.DeleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}
