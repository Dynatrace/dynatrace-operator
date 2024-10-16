package edgeconnect

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertTo(t *testing.T) {
	t.Run("migrate from edgeconnect v1alpha1 to v1alpha2", func(t *testing.T) {
		from := EdgeConnect{
			ObjectMeta: getV1alpha1Base(),
			Spec:       getV1alpha1Spec(),
			Status:     getV1Alpha1Status(),
		}
		to := edgeconnect.EdgeConnect{}

		from.ConvertTo(&to)

		assert.True(t, reflect.DeepEqual(from.ObjectMeta, to.ObjectMeta))
		toAreSpecsEqual(t, &from.Spec, &to.Spec)
		toAreStatusesEqual(t, &from.Status, &to.Status)
	})
	t.Run("migrate from edgeconnect v1alpha1 to v1alpha2 .spec.hostRestrictions is not provided", func(t *testing.T) {
		from := EdgeConnect{
			Spec: EdgeConnectSpec{},
		}
		to := edgeconnect.EdgeConnect{}

		from.ConvertTo(&to)
		assert.Nil(t, to.Spec.HostRestrictions)
	})
}

func getV1alpha1Base() metav1.ObjectMeta {
	deletionGracePeriodSeconds := int64(1)

	return metav1.ObjectMeta{
		Name:                       "a",
		GenerateName:               "b",
		Namespace:                  "c",
		SelfLink:                   "d",
		UID:                        "e",
		ResourceVersion:            "f",
		Generation:                 1,
		CreationTimestamp:          metav1.Time{Time: time.Now()},
		DeletionTimestamp:          &metav1.Time{Time: time.Now()},
		DeletionGracePeriodSeconds: &deletionGracePeriodSeconds,
		Labels: map[string]string{
			"a": "b",
		},
		Annotations: map[string]string{
			"c": "d",
		},
		OwnerReferences: nil,
		Finalizers:      nil,
		ManagedFields:   nil,
	}
}

func getV1alpha1Spec() EdgeConnectSpec {
	replicas := int32(1)

	return EdgeConnectSpec{
		Annotations: map[string]string{
			"a": "b",
		},
		Labels: map[string]string{
			"c": "d",
		},
		Replicas:     &replicas,
		NodeSelector: nil,
		KubernetesAutomation: &KubernetesAutomationSpec{
			Enabled: true,
		},
		Proxy: &ProxySpec{
			Host:    "e",
			NoProxy: "f",
			AuthRef: "g",
			Port:    1,
		},
		ImageRef: ImageRefSpec{
			Repository: "h",
			Tag:        "i",
		},
		ApiServer:          "j",
		HostRestrictions:   "k,l",
		CustomPullSecret:   "m",
		CaCertsRef:         "n",
		ServiceAccountName: "o",
		OAuth: OAuthSpec{
			ClientSecret: "p",
			Endpoint:     "q",
			Resource:     "r",
			Provisioner:  true,
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("1"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("2"),
			},
			Claims: []corev1.ResourceClaim{
				{
					Name: "s",
				},
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "t",
				Value: "u",
			},
		},
		Tolerations: []corev1.Toleration{
			{
				Key:   "v",
				Value: "w",
			},
		},
		TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
			{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key: "x",
						},
					},
				},
			},
		},
		HostPatterns: []string{
			"y",
		},
		AutoUpdate: true,
	}
}

func getV1Alpha1Status() EdgeConnectStatus {
	return EdgeConnectStatus{
		Conditions: []metav1.Condition{
			{
				Type:   "Ready",
				Status: metav1.ConditionFalse,
			},
			{
				Type:   "NotReady",
				Status: metav1.ConditionTrue,
			},
		},
		KubeSystemUID:    "a",
		DeploymentPhase:  status.Running,
		UpdatedTimestamp: metav1.Time{Time: time.Now()},
		Version: status.VersionStatus{
			LastProbeTimestamp: &metav1.Time{Time: time.Now()},
			Source:             "a",
			ImageID:            "b",
			Version:            "c",
			Type:               "d",
		},
	}
}

func toAreSpecsEqual(t *testing.T, src *EdgeConnectSpec, dst *edgeconnect.EdgeConnectSpec) {
	assert.True(t, reflect.DeepEqual(src.Annotations, dst.Annotations), "Annotations")

	assert.True(t, reflect.DeepEqual(src.Labels, dst.Labels), "Labels")

	assert.True(t, reflect.DeepEqual(src.Replicas, dst.Replicas), "Replicas")

	assert.True(t, reflect.DeepEqual(src.NodeSelector, dst.NodeSelector), "NodeSelector")

	assert.True(t, reflect.DeepEqual(src.KubernetesAutomation.Enabled, dst.KubernetesAutomation.Enabled), "KubernetesAutomation.Enabled")

	assert.True(t, reflect.DeepEqual(src.Proxy.Port, dst.Proxy.Port), "Proxy.Port")

	assert.True(t, reflect.DeepEqual(src.Proxy.NoProxy, dst.Proxy.NoProxy), "dst.Proxy.NoProxy")

	assert.True(t, reflect.DeepEqual(src.Proxy.Host, dst.Proxy.Host), "Proxy.Host")

	assert.True(t, reflect.DeepEqual(src.Proxy.AuthRef, dst.Proxy.AuthRef), "Proxy.AuthRef")

	assert.True(t, reflect.DeepEqual(src.ImageRef.Repository, dst.ImageRef.Repository), "ImageRef.Repository")

	assert.True(t, reflect.DeepEqual(src.ImageRef.Tag, dst.ImageRef.Tag), "ImageRef.Tag")

	assert.True(t, reflect.DeepEqual(src.ApiServer, dst.ApiServer), "ApiServer")

	assert.True(t, reflect.DeepEqual(strings.Split(src.HostRestrictions, ","), dst.HostRestrictions), "HostRestrictions")

	assert.True(t, reflect.DeepEqual(src.CustomPullSecret, dst.CustomPullSecret), "CustomPullSecret")

	assert.True(t, reflect.DeepEqual(src.ServiceAccountName, dst.ServiceAccountName), "ServiceAccountName")

	assert.True(t, reflect.DeepEqual(src.OAuth.Provisioner, dst.OAuth.Provisioner), "OAuth.Provisioner")

	assert.True(t, reflect.DeepEqual(src.OAuth.Endpoint, dst.OAuth.Endpoint), "OAuth.Endpoint")

	assert.True(t, reflect.DeepEqual(src.OAuth.ClientSecret, dst.OAuth.ClientSecret), "OAuth.ClientSecret")

	assert.True(t, reflect.DeepEqual(src.OAuth.Resource, dst.OAuth.Resource), "OAuth.Resource")

	assert.True(t, reflect.DeepEqual(src.Resources, dst.Resources), "Resources")

	assert.True(t, reflect.DeepEqual(src.Env, dst.Env), "Env")

	assert.True(t, reflect.DeepEqual(src.Tolerations, dst.Tolerations), "Tolerations")

	assert.True(t, reflect.DeepEqual(src.TopologySpreadConstraints, dst.TopologySpreadConstraints), "TopologySpreadConstraints")

	assert.True(t, reflect.DeepEqual(src.HostPatterns, dst.HostPatterns), "HostPatterns")

	assert.True(t, reflect.DeepEqual(src.AutoUpdate, dst.AutoUpdate), "Autoupdate")
}

func toAreStatusesEqual(t *testing.T, src *EdgeConnectStatus, dst *edgeconnect.EdgeConnectStatus) {
	assert.True(t, reflect.DeepEqual(src.Conditions, dst.Conditions), "Conditions")

	assert.True(t, reflect.DeepEqual(src.KubeSystemUID, dst.KubeSystemUID), "KubeSystemUID")

	assert.True(t, reflect.DeepEqual(src.DeploymentPhase, dst.DeploymentPhase), "DeploymentPhase")

	assert.True(t, reflect.DeepEqual(src.UpdatedTimestamp, dst.UpdatedTimestamp), "UpdatedTimestamp")

	assert.True(t, reflect.DeepEqual(src.Version, dst.Version), "Version)")
}
