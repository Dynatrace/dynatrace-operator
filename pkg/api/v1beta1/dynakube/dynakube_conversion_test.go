package dynakube

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace                 = "test-namespace"
	testName                      = "test-name"
	testUrl                       = "test-url"
	testToken                     = "test-token"
	testCustomPullSecret          = "test-custompullsecret"
	testProxyValue                = "test-proxyvalue"
	testTrustedCAs                = "test-trustedCAs"
	testNetworkZone               = "test-networkzone"
	testStatusOneAgentInstanceKey = "test-instance"
	testProfile                   = "test-profile"
)

func TestConversion_ConvertFrom_Create(t *testing.T) {
	oldDynakube := &dynakube.DynaKube{
		ObjectMeta: prepareObjectMeta(),
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			Tokens: testToken,
			OneAgent: dynakube.OneAgentSpec{
				HostMonitoring: &dynakube.HostInjectSpec{},
			},
		},
	}

	convertedDynakube := &DynaKube{
		ObjectMeta: prepareObjectMeta(),
		Spec: DynaKubeSpec{
			APIURL: testAPIURL,
			Tokens: testToken,
			MetaDataEnrichment: MetaDataEnrichment{
				Enabled: address.Of(true),
			},
			DynatraceApiRequestThreshold: address.Of(time.Duration(DefaultMinRequestThresholdMinutes)),
			OneAgent: OneAgentSpec{
				HostMonitoring: &HostInjectSpec{
					SecCompProfile: testProfile,
				},
			},
		},
	}

	err := convertedDynakube.ConvertFrom(oldDynakube)
	require.NoError(t, err)

	assert.Equal(t, oldDynakube.ObjectMeta.Namespace, convertedDynakube.ObjectMeta.Namespace)
	assert.Equal(t, oldDynakube.ObjectMeta.Name, convertedDynakube.ObjectMeta.Name)

	assert.Equal(t, oldDynakube.Spec.APIURL, convertedDynakube.Spec.APIURL)
	assert.Equal(t, oldDynakube.Spec.Tokens, convertedDynakube.Spec.Tokens)

	require.NotNil(t, convertedDynakube.Spec.MetaDataEnrichment)
	require.True(t, *convertedDynakube.Spec.MetaDataEnrichment.Enabled)
	require.NotNil(t, convertedDynakube.Spec.OneAgent.HostMonitoring)
	require.NotEmpty(t, convertedDynakube.Spec.OneAgent.HostMonitoring.SecCompProfile)
	require.NotNil(t, convertedDynakube.Spec.DynatraceApiRequestThreshold)
}

func prepareObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      testName,
	}
}

func TestConversion_ConvertFrom(t *testing.T) {
	curr_time := metav1.Now()
	oldDynakube := &dynakube.DynaKube{
		ObjectMeta: prepareObjectMeta(),
		Spec: dynakube.DynaKubeSpec{
			APIURL:           testUrl,
			Tokens:           testToken,
			CustomPullSecret: testCustomPullSecret,
			SkipCertCheck:    true,
			Proxy: &dynakube.DynaKubeProxy{
				Value: testProxyValue,
			},
			TrustedCAs:  testTrustedCAs,
			NetworkZone: testNetworkZone,
			OneAgent: dynakube.OneAgentSpec{
				HostMonitoring: &dynakube.HostInjectSpec{},
			},
		},
		Status: dynakube.DynaKubeStatus{
			Phase:            "test-phase",
			UpdatedTimestamp: curr_time,
			Conditions: []metav1.Condition{
				{
					Type:               "type",
					Status:             "status",
					ObservedGeneration: 3,
					LastTransitionTime: curr_time,
					Reason:             "reason",
					Message:            "message",
				},
			},
			OneAgent: dynakube.OneAgentStatus{
				Instances: map[string]dynakube.OneAgentInstance{
					testStatusOneAgentInstanceKey: {
						PodName:   "test-instance-podname",
						IPAddress: "test-instance-ip",
					},
				},
			},
		},
	}

	convertedDynakube := &DynaKube{
		ObjectMeta: prepareObjectMeta(),
		Spec: DynaKubeSpec{
			APIURL: testAPIURL,
			Tokens: testToken,
			MetaDataEnrichment: MetaDataEnrichment{
				Enabled: address.Of(true),
			},
			DynatraceApiRequestThreshold: address.Of(time.Duration(DefaultMinRequestThresholdMinutes)),
			OneAgent: OneAgentSpec{
				HostMonitoring: &HostInjectSpec{
					SecCompProfile: testProfile,
				},
			},
		},
	}
	err := convertedDynakube.ConvertFrom(oldDynakube)
	require.NoError(t, err)

	assert.Equal(t, oldDynakube.ObjectMeta.Namespace, convertedDynakube.ObjectMeta.Namespace)
	assert.Equal(t, oldDynakube.ObjectMeta.Name, convertedDynakube.ObjectMeta.Name)

	assert.Equal(t, oldDynakube.Spec.APIURL, convertedDynakube.Spec.APIURL)
	assert.Equal(t, oldDynakube.Spec.Tokens, convertedDynakube.Spec.Tokens)
	assert.Equal(t, oldDynakube.Spec.CustomPullSecret, convertedDynakube.Spec.CustomPullSecret)
	assert.Equal(t, oldDynakube.Spec.SkipCertCheck, convertedDynakube.Spec.SkipCertCheck)
	assert.Equal(t, oldDynakube.Spec.Proxy.ValueFrom, convertedDynakube.Spec.Proxy.ValueFrom)
	assert.Equal(t, oldDynakube.Spec.Proxy.Value, convertedDynakube.Spec.Proxy.Value)
	assert.Equal(t, oldDynakube.Spec.TrustedCAs, convertedDynakube.Spec.TrustedCAs)
	assert.Equal(t, oldDynakube.Spec.NetworkZone, convertedDynakube.Spec.NetworkZone)
	assert.Equal(t, oldDynakube.Spec.EnableIstio, convertedDynakube.Spec.EnableIstio)

	assert.Equal(t, oldDynakube.Status.Conditions, convertedDynakube.Status.Conditions)

	assert.Len(t, convertedDynakube.Status.OneAgent.Instances, 1)
	oldInstance := oldDynakube.Status.OneAgent.Instances[testStatusOneAgentInstanceKey]
	convertedInstance := convertedDynakube.Status.OneAgent.Instances[testStatusOneAgentInstanceKey]
	assert.Equal(t, oldInstance.IPAddress, convertedInstance.IPAddress)
	assert.Equal(t, oldInstance.PodName, convertedInstance.PodName)

	assert.Equal(t, oldDynakube.Status.OneAgent.Version, convertedDynakube.Status.OneAgent.Version)
	assert.Equal(t, string(oldDynakube.Status.Phase), string(convertedDynakube.Status.Phase))
	assert.Equal(t, oldDynakube.Status.UpdatedTimestamp, convertedDynakube.Status.UpdatedTimestamp)

	require.NotNil(t, convertedDynakube.Spec.MetaDataEnrichment)
	require.True(t, *convertedDynakube.Spec.MetaDataEnrichment.Enabled)
	require.NotNil(t, convertedDynakube.Spec.OneAgent.HostMonitoring)
	require.NotEmpty(t, convertedDynakube.Spec.OneAgent.HostMonitoring.SecCompProfile)
	require.NotNil(t, convertedDynakube.Spec.DynatraceApiRequestThreshold)
}

func TestConversion_ConvertTo(t *testing.T) {
	timeNow := metav1.Now()
	oldDynakube := &DynaKube{
		ObjectMeta: prepareObjectMeta(),
		Spec: DynaKubeSpec{
			APIURL:           testUrl,
			Tokens:           testToken,
			CustomPullSecret: testCustomPullSecret,
			SkipCertCheck:    true,
			Proxy: &DynaKubeProxy{
				Value: testProxyValue,
			},
			TrustedCAs:                   testTrustedCAs,
			NetworkZone:                  testNetworkZone,
			DynatraceApiRequestThreshold: address.Of(time.Duration(DefaultMinRequestThresholdMinutes)),
			OneAgent:                     OneAgentSpec{},
			MetaDataEnrichment: MetaDataEnrichment{
				Enabled: address.Of(true),
			},
		},
		Status: DynaKubeStatus{
			Phase:                   "test-phase",
			UpdatedTimestamp:        timeNow,
			LastTokenProbeTimestamp: &timeNow,
			Conditions: []metav1.Condition{
				{
					Type:               "type",
					Status:             "status",
					ObservedGeneration: 3,
					LastTransitionTime: timeNow,
					Reason:             "reason",
					Message:            "message",
				},
			},
			OneAgent: OneAgentStatus{
				Instances: map[string]OneAgentInstance{
					testStatusOneAgentInstanceKey: {
						PodName:   "test-instance-podname",
						IPAddress: "test-instance-ip",
					},
				},
			},
		},
	}

	convertedDynakube := &dynakube.DynaKube{}

	err := oldDynakube.ConvertTo(convertedDynakube)
	require.NoError(t, err)

	assert.Equal(t, oldDynakube.ObjectMeta.Namespace, convertedDynakube.ObjectMeta.Namespace)
	assert.Equal(t, oldDynakube.ObjectMeta.Name, convertedDynakube.ObjectMeta.Name)

	assert.Equal(t, oldDynakube.Spec.APIURL, convertedDynakube.Spec.APIURL)
	assert.Equal(t, oldDynakube.Spec.Tokens, convertedDynakube.Spec.Tokens)
	assert.Equal(t, oldDynakube.Spec.CustomPullSecret, convertedDynakube.Spec.CustomPullSecret)
	assert.Equal(t, oldDynakube.Spec.SkipCertCheck, convertedDynakube.Spec.SkipCertCheck)
	assert.Equal(t, oldDynakube.Spec.Proxy.ValueFrom, convertedDynakube.Spec.Proxy.ValueFrom)
	assert.Equal(t, oldDynakube.Spec.Proxy.Value, convertedDynakube.Spec.Proxy.Value)
	assert.Equal(t, oldDynakube.Spec.TrustedCAs, convertedDynakube.Spec.TrustedCAs)
	assert.Equal(t, oldDynakube.Spec.NetworkZone, convertedDynakube.Spec.NetworkZone)
	assert.Equal(t, oldDynakube.Spec.EnableIstio, convertedDynakube.Spec.EnableIstio)

	require.NotNil(t, convertedDynakube.Spec.ActiveGate)
	assert.Equal(t, oldDynakube.Spec.ActiveGate.Image, convertedDynakube.Spec.ActiveGate.Image)

	assert.Equal(t, oldDynakube.Status.Conditions, convertedDynakube.Status.Conditions)

	assert.Len(t, convertedDynakube.Status.OneAgent.Instances, 1)
	oldInstance := oldDynakube.Status.OneAgent.Instances[testStatusOneAgentInstanceKey]
	convertedInstance := convertedDynakube.Status.OneAgent.Instances[testStatusOneAgentInstanceKey]
	assert.Equal(t, oldInstance.IPAddress, convertedInstance.IPAddress)
	assert.Equal(t, oldInstance.PodName, convertedInstance.PodName)

	assert.Equal(t, oldDynakube.Status.OneAgent.Version, convertedDynakube.Status.OneAgent.Version)
	assert.Equal(t, string(oldDynakube.Status.Phase), string(convertedDynakube.Status.Phase))
	assert.Equal(t, oldDynakube.Status.UpdatedTimestamp, convertedDynakube.Status.UpdatedTimestamp)
}
