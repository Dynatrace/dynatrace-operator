package edgeconnect

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/edgeconnect"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertTo(t *testing.T) {
	t.Run("migrate from edgeconnect v1alpha1 to v1beta1", func(t *testing.T) {
		from := EdgeConnect{
			ObjectMeta: getV1alpha1Base(),
			Spec:       getV1alpha1Spec(),
			Status:     getV1Alpha1Status(),
		}
		to := edgeconnect.EdgeConnect{}

		from.ConvertTo(&to)

		assert.True(t, reflect.DeepEqual(from.ObjectMeta, to.ObjectMeta))
		assert.True(t, toAreSpecsEqual(&from.Spec, &to.Spec))
		assert.True(t, toAreStatusesEqual(&from.Status, &to.Status))
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

func toAreSpecsEqual(src *EdgeConnectSpec, dst *edgeconnect.EdgeConnectSpec) bool { //nolint:revive
	if !reflect.DeepEqual(src.Annotations, dst.Annotations) {
		return false
	}

	if !reflect.DeepEqual(src.Labels, dst.Labels) {
		return false
	}

	if !reflect.DeepEqual(src.Replicas, dst.Replicas) {
		return false
	}

	if !reflect.DeepEqual(src.NodeSelector, dst.NodeSelector) {
		return false
	}

	if !reflect.DeepEqual(src.KubernetesAutomation.Enabled, dst.KubernetesAutomation.Enabled) {
		return false
	}

	if !reflect.DeepEqual(src.Proxy.Port, dst.Proxy.Port) {
		return false
	}

	if !reflect.DeepEqual(src.Proxy.NoProxy, dst.Proxy.NoProxy) {
		return false
	}

	if !reflect.DeepEqual(src.Proxy.Host, dst.Proxy.Host) {
		return false
	}

	if !reflect.DeepEqual(src.Proxy.AuthRef, dst.Proxy.AuthRef) {
		return false
	}

	if !reflect.DeepEqual(src.ImageRef.Repository, dst.ImageRef.Repository) {
		return false
	}

	if !reflect.DeepEqual(src.ImageRef.Tag, dst.ImageRef.Tag) {
		return false
	}

	if !reflect.DeepEqual(src.ApiServer, dst.ApiServer) {
		return false
	}

	if !reflect.DeepEqual(strings.Split(src.HostRestrictions, ","), dst.HostRestrictions) {
		return false
	}

	if !reflect.DeepEqual(src.CustomPullSecret, dst.CustomPullSecret) {
		return false
	}

	if !reflect.DeepEqual(src.ServiceAccountName, dst.ServiceAccountName) {
		return false
	}

	if !reflect.DeepEqual(src.OAuth.Provisioner, dst.OAuth.Provisioner) {
		return false
	}

	if !reflect.DeepEqual(src.OAuth.Endpoint, dst.OAuth.Endpoint) {
		return false
	}

	if !reflect.DeepEqual(src.OAuth.ClientSecret, dst.OAuth.ClientSecret) {
		return false
	}

	if !reflect.DeepEqual(src.OAuth.Resource, dst.OAuth.Resource) {
		return false
	}

	if !reflect.DeepEqual(src.Resources, dst.Resources) {
		return false
	}

	if !reflect.DeepEqual(src.Env, dst.Env) {
		return false
	}

	if !reflect.DeepEqual(src.Tolerations, dst.Tolerations) {
		return false
	}

	if !reflect.DeepEqual(src.TopologySpreadConstraints, dst.TopologySpreadConstraints) {
		return false
	}

	if !reflect.DeepEqual(src.HostPatterns, dst.HostPatterns) {
		return false
	}

	if !reflect.DeepEqual(src.AutoUpdate, dst.AutoUpdate) {
		return false
	}

	return true
}

func toAreStatusesEqual(src *EdgeConnectStatus, dst *edgeconnect.EdgeConnectStatus) bool {
	if !reflect.DeepEqual(src.Conditions, dst.Conditions) {
		return false
	}

	if !reflect.DeepEqual(src.KubeSystemUID, dst.KubeSystemUID) {
		return false
	}

	if !reflect.DeepEqual(src.DeploymentPhase, dst.DeploymentPhase) {
		return false
	}

	if !reflect.DeepEqual(src.UpdatedTimestamp, dst.UpdatedTimestamp) {
		return false
	}

	if !reflect.DeepEqual(src.Version, dst.Version) {
		return false
	}

	return true
}
