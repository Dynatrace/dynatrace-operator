package oneagent

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/move"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/mounts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/volumes"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestMutateInitContainer(t *testing.T) {
	installPath := "test/path"
	initContainerBase := corev1.Container{
		Args: []string{
			bootstrapper.Use,
		},
		Image: "webhook-image",
		Resources: corev1.ResourceRequirements{
			Requests: resources.NewResourceList("30m", "30Mi"), // add some defaults
		},
	}

	t.Run("csi-scenario -> custom init-resources", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: true})

		dk := dynakube.DynaKube{}
		dk.Name = "csi-scenario"
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		dk.Spec.OneAgent.ApplicationMonitoring.InitResources = &corev1.ResourceRequirements{
			Requests: resources.NewResourceList("40m", "40Mi"),
		}
		pod := &corev1.Pod{}

		request := &webhook.MutationRequest{
			BaseRequest: &webhook.BaseRequest{
				Pod:      pod,
				DynaKube: dk,
			},
			InstallContainer: initContainerBase.DeepCopy(),
		}

		err := mutateInitContainer(request, installPath)
		require.NoError(t, err)

		csiVolume, err := volumes.GetByName(request.Pod.Spec.Volumes, BinVolumeName)
		require.NoError(t, err)
		require.NotNil(t, csiVolume.CSI)
		require.NotNil(t, csiVolume.CSI.ReadOnly)
		require.True(t, *csiVolume.CSI.ReadOnly)

		csiMount, err := mounts.GetByName(request.InstallContainer.VolumeMounts, BinVolumeName)
		require.NoError(t, err)
		require.True(t, csiMount.ReadOnly)

		assert.NotEmpty(t, request.InstallContainer.Args)
		assert.Subset(t, request.InstallContainer.Args, initContainerBase.Args)
		assert.Equal(t, initContainerBase.Image, request.InstallContainer.Image)
		assert.Equal(t, *dk.Spec.OneAgent.ApplicationMonitoring.InitResources, request.InstallContainer.Resources) // respects custom resources
	})

	t.Run("csi-scenario", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: true})

		dk := dynakube.DynaKube{}
		dk.Name = "csi-scenario"
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		pod := &corev1.Pod{}

		request := &webhook.MutationRequest{
			BaseRequest: &webhook.BaseRequest{
				Pod:      pod,
				DynaKube: dk,
			},
			InstallContainer: initContainerBase.DeepCopy(),
		}

		err := mutateInitContainer(request, installPath)
		require.NoError(t, err)

		csiVolume, err := volumes.GetByName(request.Pod.Spec.Volumes, BinVolumeName)
		require.NoError(t, err)
		require.NotNil(t, csiVolume.CSI)
		require.NotNil(t, csiVolume.CSI.ReadOnly)
		require.True(t, *csiVolume.CSI.ReadOnly)

		csiMount, err := mounts.GetByName(request.InstallContainer.VolumeMounts, BinVolumeName)
		require.NoError(t, err)
		require.True(t, csiMount.ReadOnly)

		assert.NotEmpty(t, request.InstallContainer.Args)
		assert.Subset(t, request.InstallContainer.Args, initContainerBase.Args)
		assert.Equal(t, initContainerBase.Image, request.InstallContainer.Image)
		assert.NotEmpty(t, request.InstallContainer.Resources) // does not touch default
	})

	t.Run("zip-scenario", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		flavor := "musl"
		version := "1.2.3"
		dk := dynakube.DynaKube{}
		dk.Name = "zip-scenario"
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		dk.Status.CodeModules.Version = version
		pod := &corev1.Pod{}
		pod.Annotations = map[string]string{
			AnnotationFlavor: flavor,
		}

		request := &webhook.MutationRequest{
			BaseRequest: &webhook.BaseRequest{
				Pod:      pod,
				DynaKube: dk,
			},
			InstallContainer: initContainerBase.DeepCopy(),
		}

		err := mutateInitContainer(request, installPath)
		require.NoError(t, err)

		emptyDirVolume, err := volumes.GetByName(request.Pod.Spec.Volumes, BinVolumeName)
		require.NoError(t, err)
		require.NotNil(t, emptyDirVolume.EmptyDir)

		emptyDirMount, err := mounts.GetByName(request.InstallContainer.VolumeMounts, BinVolumeName)
		require.NoError(t, err)
		require.False(t, emptyDirMount.ReadOnly)

		assert.NotEmpty(t, request.InstallContainer.Args)
		assert.Subset(t, request.InstallContainer.Args, initContainerBase.Args)
		assert.Contains(t, request.InstallContainer.Args, fmt.Sprintf("--%s=%s", bootstrapper.FlavorFlag, flavor))
		assert.Contains(t, request.InstallContainer.Args, fmt.Sprintf("--%s=%s", bootstrapper.TargetVersionFlag, version))
		assert.Equal(t, initContainerBase.Image, request.InstallContainer.Image)

		assert.Empty(t, request.InstallContainer.Resources) // removes default, as they wouldn't work
	})

	t.Run("node-image-pull-scenario", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		image := "myimage.io:latest"
		dk := dynakube.DynaKube{}
		dk.Name = "node-image-pull-scenario"
		dk.Annotations = map[string]string{
			exp.OANodeImagePullKey: "true",
		}
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		dk.Status.CodeModules.ImageID = image
		pod := &corev1.Pod{}

		request := &webhook.MutationRequest{
			BaseRequest: &webhook.BaseRequest{
				Pod:      pod,
				DynaKube: dk,
			},
			InstallContainer: initContainerBase.DeepCopy(),
		}

		err := mutateInitContainer(request, installPath)
		require.NoError(t, err)

		emptyDirVolume, err := volumes.GetByName(request.Pod.Spec.Volumes, BinVolumeName)
		require.NoError(t, err)
		require.NotNil(t, emptyDirVolume.EmptyDir)

		emptyDirMount, err := mounts.GetByName(request.InstallContainer.VolumeMounts, BinVolumeName)
		require.NoError(t, err)
		require.False(t, emptyDirMount.ReadOnly)

		assert.NotEmpty(t, request.InstallContainer.Args)
		assert.NotSubset(t, request.InstallContainer.Args, initContainerBase.Args)
		assert.Equal(t, image, request.InstallContainer.Image)

		assert.Empty(t, request.InstallContainer.Resources) // removes default, as they wouldn't work
	})

	t.Run("zip-scenario -> custom init-resources", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		flavor := "musl"
		version := "1.2.3"
		dk := dynakube.DynaKube{}
		dk.Name = "zip-scenario"
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		dk.Spec.OneAgent.ApplicationMonitoring.InitResources = &corev1.ResourceRequirements{
			Requests: resources.NewResourceList("40m", "40Mi"),
		}
		dk.Status.CodeModules.Version = version
		pod := &corev1.Pod{}
		pod.Annotations = map[string]string{
			AnnotationFlavor: flavor,
		}

		request := &webhook.MutationRequest{
			BaseRequest: &webhook.BaseRequest{
				Pod:      pod,
				DynaKube: dk,
			},
			InstallContainer: initContainerBase.DeepCopy(),
		}

		err := mutateInitContainer(request, installPath)
		require.NoError(t, err)

		assert.Equal(t, *dk.Spec.OneAgent.ApplicationMonitoring.InitResources, request.InstallContainer.Resources) // respects custom resources
	})

	t.Run("node-image-pull-scenario -> custom init-resources", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		image := "myimage.io:latest"
		dk := dynakube.DynaKube{}
		dk.Name = "node-image-pull-scenario"
		dk.Annotations = map[string]string{
			exp.OANodeImagePullKey: "true",
		}
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
		dk.Spec.OneAgent.ApplicationMonitoring.InitResources = &corev1.ResourceRequirements{
			Requests: resources.NewResourceList("40m", "40Mi"),
		}
		dk.Status.CodeModules.ImageID = image
		pod := &corev1.Pod{}

		request := &webhook.MutationRequest{
			BaseRequest: &webhook.BaseRequest{
				Pod:      pod,
				DynaKube: dk,
			},
			InstallContainer: initContainerBase.DeepCopy(),
		}

		err := mutateInitContainer(request, installPath)
		require.NoError(t, err)

		assert.Equal(t, *dk.Spec.OneAgent.ApplicationMonitoring.InitResources, request.InstallContainer.Resources) // respects custom resources
	})
}

func TestAddInitArgs(t *testing.T) {
	installPath := "test/install"
	commonArgs := []string{
		fmt.Sprintf("--%s=%s", cmd.SourceFolderFlag, AgentCodeModuleSource),
		fmt.Sprintf("--%s=%s", cmd.TargetFolderFlag, consts.AgentInitBinDirMount),
		fmt.Sprintf("--%s=%s", configure.InstallPathFlag, installPath),
	}

	t.Run("default appmon -> only common args", func(t *testing.T) {
		pod := corev1.Pod{}
		dk := dynakube.DynaKube{}
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}

		initContainer := corev1.Container{}

		err := addInitArgs(&pod, &initContainer, dk, installPath)
		require.NoError(t, err)

		assert.ElementsMatch(t, commonArgs, initContainer.Args)
	})
	t.Run("default cloudnative -> common args + cloudnative args", func(t *testing.T) {
		tenantUUID := "my-tenant-123"
		pod := corev1.Pod{}
		dk := dynakube.DynaKube{}
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		dk.Status.ActiveGate.ConnectionInfo.TenantUUID = tenantUUID

		initContainer := corev1.Container{}

		err := addInitArgs(&pod, &initContainer, dk, installPath)
		require.NoError(t, err)

		expectedArgs := []string{
			fmt.Sprintf("--%s=%s", configure.TenantFlag, tenantUUID),
			fmt.Sprintf("--%s", configure.IsFullstackFlag),
		}
		expectedArgs = append(expectedArgs, commonArgs...)

		assert.ElementsMatch(t, expectedArgs, initContainer.Args)
	})
	t.Run("default cloudnative + no-tenant -> error", func(t *testing.T) {
		pod := corev1.Pod{}
		dk := dynakube.DynaKube{}
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		dk.Status.ActiveGate.ConnectionInfo.TenantUUID = ""

		initContainer := corev1.Container{}

		err := addInitArgs(&pod, &initContainer, dk, installPath)
		require.ErrorAs(t, err, new(webhook.MutatorError))
	})
	t.Run("cloudnative + tech from dk -> common args + cloudnative args + tech arg", func(t *testing.T) {
		tenantUUID := "my-tenant-123"
		technology := "java,php"
		pod := corev1.Pod{}
		dk := dynakube.DynaKube{}
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		dk.Status.ActiveGate.ConnectionInfo.TenantUUID = tenantUUID
		dk.Annotations = map[string]string{
			AnnotationTechnologies: technology,
		}

		initContainer := corev1.Container{}

		err := addInitArgs(&pod, &initContainer, dk, installPath)
		require.NoError(t, err)

		expectedArgs := []string{
			fmt.Sprintf("--%s=%s", move.TechnologyFlag, technology),
			fmt.Sprintf("--%s=%s", configure.TenantFlag, tenantUUID),
			fmt.Sprintf("--%s", configure.IsFullstackFlag),
		}
		expectedArgs = append(expectedArgs, commonArgs...)

		assert.ElementsMatch(t, expectedArgs, initContainer.Args)
	})

	t.Run("appmon + tech from pod -> common args + tech arg", func(t *testing.T) {
		technology := "java,php"
		pod := corev1.Pod{}
		pod.Annotations = map[string]string{
			AnnotationTechnologies: technology,
		}
		dk := dynakube.DynaKube{}
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}

		initContainer := corev1.Container{}

		err := addInitArgs(&pod, &initContainer, dk, installPath)
		require.NoError(t, err)

		expectedArgs := []string{
			fmt.Sprintf("--%s=%s", move.TechnologyFlag, technology),
		}
		expectedArgs = append(expectedArgs, commonArgs...)

		assert.ElementsMatch(t, expectedArgs, initContainer.Args)
	})
}

func TestGetTechnology(t *testing.T) {
	type testCase struct {
		title          string
		podAnnotations map[string]string
		dkAnnotations  map[string]string
		expected       string
	}

	testCases := []testCase{
		{
			title:          "nil annotations -> empty string",
			podAnnotations: nil,
			dkAnnotations:  nil,
			expected:       "",
		},
		{
			title:          "empty annotations -> empty string",
			podAnnotations: map[string]string{},
			dkAnnotations:  map[string]string{},
			expected:       "",
		},
		{
			title:          "from dk",
			podAnnotations: map[string]string{},
			dkAnnotations: map[string]string{
				AnnotationTechnologies: "java,php",
			},
			expected: "java,php",
		},
		{
			title: "from pod",
			podAnnotations: map[string]string{
				AnnotationTechnologies: "java,php",
			},
			dkAnnotations: map[string]string{},
			expected:      "java,php",
		},
		{
			title: "pod overrules dk",
			podAnnotations: map[string]string{
				AnnotationTechnologies: "java,php",
			},
			dkAnnotations: map[string]string{
				AnnotationTechnologies: "overruled,value",
			},
			expected: "java,php",
		},
	}

	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			pod := corev1.Pod{}
			dk := dynakube.DynaKube{}

			pod.Annotations = test.podAnnotations
			dk.Annotations = test.dkAnnotations

			assert.Equal(t, test.expected, getTechnology(pod, dk))
		})
	}
}
