package dynakube

import (
	"testing"
	_ "time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
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
	testOneAgentImage             = "test-oneagent-image"
	testOneAgentVersion           = "test-oneagent-version"
	testPriorityClassName         = "test-priorityclassname"
	testDNSPolicy                 = "test-dnspolicy"
	testActiveGateImage           = "test-activegateimage"
	testStatusOneAgentInstanceKey = "test-instance"
)

func TestConversion_ConvertTFrom_Create(t *testing.T) {
	oldDynakube := &dynakube.DynaKube{
		ObjectMeta: prepareObjectMeta(),
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			Tokens: testToken,
		},
	}

	oldDynakube.Annotations = map[string]string{
		dynakube.AnnotationFeatureApiRequestThreshold:    "",
		dynakube.AnnotationFeatureOneAgentSecCompProfile: "test",
		dynakube.AnnotationFeatureMetadataEnrichment:     "true",
	}
	convertedDynakube := &DynaKube{}
	err := convertedDynakube.ConvertFrom(oldDynakube)
	require.NoError(t, err)

	assert.Equal(t, oldDynakube.ObjectMeta.Namespace, convertedDynakube.ObjectMeta.Namespace)
	assert.Equal(t, oldDynakube.ObjectMeta.Name, convertedDynakube.ObjectMeta.Name)

	assert.Equal(t, oldDynakube.Spec.APIURL, convertedDynakube.Spec.APIURL)
	assert.Equal(t, oldDynakube.Spec.Tokens, convertedDynakube.Spec.Tokens)

	require.NotNil(t, convertedDynakube.Spec.MetaDataEnrichment)
	require.True(t, convertedDynakube.Spec.MetaDataEnrichment.Enabled)
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
