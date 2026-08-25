// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package edgeconnect

import (
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertFrom(t *testing.T) {
	t.Run("migrate from edgeconnect v1alpha2 to v1alpha1", func(t *testing.T) {
		from := edgeconnect.EdgeConnect{
			ObjectMeta: testGetCurrentBase(),
			Spec:       testGetCurrentSpec(),
			Status:     testGetCurrentStatus(),
		}
		to := EdgeConnect{}

		require.NoError(t, to.ConvertFrom(&from))

		assert.Equal(t, from.ObjectMeta, to.ObjectMeta)
		testFromAreSpecsEqual(t, &from.Spec, &to.Spec)
		testFromAreStatusesEqual(t, &from.Status, &to.Status)
	})
}

func testGetCurrentBase() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:                       "a",
		GenerateName:               "b",
		Namespace:                  "c",
		UID:                        "d",
		ResourceVersion:            "e",
		Generation:                 1,
		CreationTimestamp:          metav1.Time{Time: time.Now()},
		DeletionTimestamp:          &metav1.Time{Time: time.Now()},
		DeletionGracePeriodSeconds: new(int64(1)),
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

func testGetCurrentSpec() edgeconnect.EdgeConnectSpec {
	return edgeconnect.EdgeConnectSpec{
		Annotations: map[string]string{
			"a": "b",
		},
		Labels: map[string]string{
			"c": "d",
		},
		Replicas:     new(int32(1)),
		NodeSelector: nil,
		KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{
			Enabled: true,
		},
		Proxy: &proxy.Spec{
			Host:    "e",
			NoProxy: "f",
			AuthRef: "g",
			Port:    1,
		},
		ImageRef: image.Ref{
			Repository: "h",
			Tag:        "i",
		},
		APIServer: "j",
		HostRestrictions: []string{
			"k",
			"l",
		},
		CustomPullSecret:   "m",
		CaCertsRef:         "n",
		ServiceAccountName: new("o"),
		OAuth: edgeconnect.OAuthSpec{
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
		AutoUpdate: new(true),
	}
}

func testGetCurrentStatus() edgeconnect.EdgeConnectStatus {
	return edgeconnect.EdgeConnectStatus{
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

func testFromAreSpecsEqual(t *testing.T, src *edgeconnect.EdgeConnectSpec, dst *EdgeConnectSpec) {
	t.Helper()

	assert.Equal(t, src.Annotations, dst.Annotations, "Annotations")

	assert.Equal(t, src.Labels, dst.Labels, "Labels")

	assert.Equal(t, src.Replicas, dst.Replicas, "Replicas")

	assert.Equal(t, src.NodeSelector, dst.NodeSelector, "NodeSelector")

	assert.Equal(t, src.KubernetesAutomation.Enabled, dst.KubernetesAutomation.Enabled, "KubernetesAutomation.Enabled")

	assert.Equal(t, src.Proxy.Port, dst.Proxy.Port, "Proxy.Port")

	assert.Equal(t, src.Proxy.NoProxy, dst.Proxy.NoProxy, "Proxy.NoProxy")

	assert.Equal(t, src.Proxy.Host, dst.Proxy.Host, "Proxy.Host")

	assert.Equal(t, src.Proxy.AuthRef, dst.Proxy.AuthRef, "Proxy.AuthRef")

	assert.Equal(t, src.ImageRef.Repository, dst.ImageRef.Repository, "ImageRef.Repository")

	assert.Equal(t, src.ImageRef.Tag, dst.ImageRef.Tag, "ImageRef.Tag")

	assert.Equal(t, src.APIServer, dst.APIServer, "ApiServer")

	if len(src.HostRestrictions) == 0 {
		assert.Empty(t, dst.HostRestrictions, "HostRestrictions")
	} else {
		assert.Equal(t, src.HostRestrictions, strings.Split(dst.HostRestrictions, ","), "HostRestrictions")
	}

	assert.Equal(t, src.CustomPullSecret, dst.CustomPullSecret, "CustomPullSecret")

	assert.Equal(t, *src.ServiceAccountName, dst.ServiceAccountName, "ServiceAccountName")

	assert.Equal(t, src.OAuth.Provisioner, dst.OAuth.Provisioner, "OAuth.Provisioner")

	assert.Equal(t, src.OAuth.Endpoint, dst.OAuth.Endpoint, "OAuth.Endpoint")

	assert.Equal(t, src.OAuth.ClientSecret, dst.OAuth.ClientSecret, "OAuth.ClientSecret")

	assert.Equal(t, src.OAuth.Resource, dst.OAuth.Resource, "OAuth.Resource")

	assert.Equal(t, src.Resources, dst.Resources, "Resources")

	assert.Equal(t, src.Env, dst.Env, "Env")

	assert.Equal(t, src.Tolerations, dst.Tolerations, "Tolerations")

	assert.Equal(t, src.TopologySpreadConstraints, dst.TopologySpreadConstraints, "TopologySpreadConstraints")

	assert.Equal(t, src.HostPatterns, dst.HostPatterns, "HostPatterns")

	assert.Equal(t, *src.AutoUpdate, dst.AutoUpdate, "AutoUpdate")
}

func testFromAreStatusesEqual(t *testing.T, src *edgeconnect.EdgeConnectStatus, dst *EdgeConnectStatus) {
	t.Helper()

	assert.Equal(t, src.Conditions, dst.Conditions, "Conditions")

	assert.Equal(t, src.KubeSystemUID, dst.KubeSystemUID, "KubeSystemUID")

	assert.Equal(t, src.DeploymentPhase, dst.DeploymentPhase, "DeploymentPhase")

	assert.Equal(t, src.UpdatedTimestamp, dst.UpdatedTimestamp, "UpdatedTimestamp")

	assert.Equal(t, src.Version, dst.Version, "Version")
}
