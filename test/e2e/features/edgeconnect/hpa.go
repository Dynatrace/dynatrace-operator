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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	hpaScaleReplicas = ptr.To(int32(3))
	hpaBaseReplicas  = ptr.To(int32(2))
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

	builder.Assess("check if EC doesn't have any replica count set", ecComponents.WaitForReplicas(testEdgeConnect, nil))
	builder.Assess("check if the EC deployment has replicas set to 1", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, 1))

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
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if EC doesn't have any replica count set", ecComponents.WaitForReplicas(testEdgeConnect, nil))
	builder.Assess("check if the EC deployment has replicas autoscaled to 3", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *hpaScaleReplicas))

	builder.Teardown(k8shpa.Delete(testHPA))
	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(ecComponents.DeleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}

func WithHPAEnforceReplicasRegular(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-with-hpa-enforce-replicas-regular")

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
		ecComponents.WithReplicas(hpaBaseReplicas),
	)

	// create OAuth client secret related to the specific EdgeConnect configuration on the tenant
	builder.Assess("create client secret", tenant.CreateClientSecret(&edgeConnectTenantConfig.Secret, ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	ecComponents.Install(builder, helpers.LevelAssess, nil, testEdgeConnect)

	builder.Assess("check if EC has replica count set to 2", ecComponents.WaitForReplicas(testEdgeConnect, hpaBaseReplicas))
	builder.Assess("check if the EC deployment has replicas set to 2", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *hpaBaseReplicas))

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
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if HPA updated the replica count", k8shpa.WaitForCurrentReplicas(testHPA, *hpaScaleReplicas))
	builder.Assess("check if EC still has replicas set to 2", ecComponents.WaitForReplicas(testEdgeConnect, hpaBaseReplicas))
	builder.Assess("check if the EC deployment replica count 2 is enforced", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *hpaBaseReplicas))

	testEdgeConnect.Spec.Replicas = nil
	builder.Assess("remove enforced replicas", ecComponents.Update(testEdgeConnect))
	builder.Assess("check if the EC deployment was autoscaled to 3", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *hpaScaleReplicas))

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

	builder.Assess("check if EC doesn't have any replica count set", ecComponents.WaitForReplicas(testEdgeConnect, nil))
	builder.Assess("check if the EC deployment has replicas set to 1", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, 1))

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
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if EC doesn't have any replica count set", ecComponents.WaitForReplicas(testEdgeConnect, nil))
	builder.Assess("check if the EC deployment has replicas autoscaled to 3", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *hpaScaleReplicas))

	builder.Teardown(k8shpa.Delete(testHPA))
	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(ecComponents.DeleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}

func WithHPAEnforceReplicasProvisioner(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-with-hpa-enforce-replicas-provisioner")

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
		ecComponents.WithReplicas(hpaBaseReplicas),
	)

	ecComponents.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))

	builder.Assess("check if EC has replica count set to 2", ecComponents.WaitForReplicas(testEdgeConnect, hpaBaseReplicas))
	builder.Assess("check if the EC deployment has replicas set to 2", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *hpaBaseReplicas))

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
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if HPA updated the replica count", k8shpa.WaitForCurrentReplicas(testHPA, *hpaScaleReplicas))
	builder.Assess("check if EC still has replicas set to 2", ecComponents.WaitForReplicas(testEdgeConnect, hpaBaseReplicas))
	builder.Assess("check if the EC deployment replica count 2 is enforced by EC", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *hpaBaseReplicas))

	testEdgeConnect.Spec.Replicas = nil
	builder.Assess("remove enforced replicas", ecComponents.Update(testEdgeConnect))
	builder.Assess("check if the EC deployment was autoscaled to 3", k8sdeployment.WaitForReplicas(testEdgeConnect.Name, testEdgeConnect.Namespace, *hpaScaleReplicas))

	builder.Teardown(k8shpa.Delete(testHPA))
	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(ecComponents.DeleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}
