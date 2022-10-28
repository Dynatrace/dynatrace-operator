package troubleshoot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImagePullableActiveGateEndpoint(t *testing.T) {
	t.Run("ActiveGate image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
		}
		endpoint := getActiveGateImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidImageNameWithoutVersion, endpoint, "invalid image")
	})
	t.Run("ActiveGate custom image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).
				withApiUrl(testApiUrl).
				withActiveGateImage(testValidCustomImageNameWithVersion).
				withActiveGateCapability("routing").
				build(),
		}
		endpoint := getActiveGateImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidCustomImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("ActiveGate custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).
				withApiUrl(testApiUrl).
				withActiveGateImage(testValidCustomImageNameWithoutVersion).
				withActiveGateCapability("routing").
				build(),
		}
		endpoint := getActiveGateImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidCustomImageNameWithoutVersion, endpoint, "invalid image")
	})
}

func TestImagePullableOneAgentEndpoint(t *testing.T) {
	t.Run("OneAgent image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithoutVersion, endpoint, "invalid image")
	})

	t.Run("Classic Full Stack OneAgent custom image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackCustomImage(testValidOneAgentCustomImageNameWithVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Classic Full Stack OneAgent custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackCustomImage(testValidOneAgentCustomImageNameWithoutVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})
	t.Run("Classic Full Stack OneAgent regular image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Classic Full Stack OneAgent custom image and Version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withClassicFullStackCustomImage(testValidOneAgentCustomImageNameWithoutVersion).withClassicFullStackImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})

	t.Run("Cloud Native OneAgent custom image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackCustomImage(testValidOneAgentCustomImageNameWithVersion).build(),
		}

		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Cloud Native OneAgent custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackCustomImage(testValidOneAgentCustomImageNameWithoutVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})
	t.Run("Cloud Native OneAgent regular image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Classic Full Stack OneAgent custom image and Version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeFullStackCustomImage(testValidOneAgentCustomImageNameWithoutVersion).withCloudNativeFullStackImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})

	t.Run("Host Monitoring OneAgent custom image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringCustomImage(testValidOneAgentImageNameWithVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("Host Monitoring OneAgent custom image without version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringCustomImage(testValidOneAgentImageNameWithoutVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithoutVersion, endpoint, "invalid image")
	})
	t.Run("Host Monitoring OneAgent regular image with version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentImageNameWithVersion, endpoint, "invalid image")
	})
	t.Run("HostMonitoring OneAgent custom image and Version", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withHostMonitoringCustomImage(testValidOneAgentCustomImageNameWithoutVersion).withHostMonitoringImageVersion(testVersion).build(),
		}
		endpoint := getOneAgentImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCustomImageNameWithoutVersion, endpoint, "invalid image")
	})
}

func TestImagePullableOneAgentCodeModulesEndpoint(t *testing.T) {
	t.Run("CloudNative codeModules image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).withCloudNativeCodeModulesImage(testValidOneAgentCodeModulesImageName).build(),
		}
		endpoint := getOneAgentCodeModulesImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCodeModulesImageName, endpoint, "invalid image")
	})
	t.Run("ApplicationMonitoring codeModules image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube: *testNewDynakubeBuilder(testNamespace, testDynakube).
				withApiUrl(testApiUrl).
				withApplicationMonitoringCodeModulesImage(testValidOneAgentCodeModulesImageName).
				withApplicationMonitoringUseCSIDriver(true).
				build(),
		}
		endpoint := getOneAgentCodeModulesImageEndpoint(&troubleshootCtx)
		assert.Equal(t, testValidOneAgentCodeModulesImageName, endpoint, "invalid image")
	})
	t.Run("No codeModules image", func(t *testing.T) {
		troubleshootCtx := troubleshootContext{
			namespaceName: testNamespace,
			dynakubeName:  testDynakube,
			dynakube:      *testNewDynakubeBuilder(testNamespace, testDynakube).withApiUrl(testApiUrl).build(),
		}
		endpoint := getOneAgentCodeModulesImageEndpoint(&troubleshootCtx)
		assert.Equal(t, "", endpoint, "expected empty image")
	})
}
