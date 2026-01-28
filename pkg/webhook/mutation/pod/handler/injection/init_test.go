package injection

import (
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smount"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8svolume"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testWebhookImage  = "test-wh-image"
	testNamespaceName = "test-namespace"
	testPodName       = "test-pod"
	testDynakubeName  = "test-dynakube"
	testUser          = int64(420)
)

func TestCreateInitContainerBase(t *testing.T) {
	wh := createTestHandler(webhookmock.NewMutator(t), webhookmock.NewMutator(t))

	t.Run("should create the init container with set container sec ctx but without user and group", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext.RunAsUser = nil
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = nil

		initContainer := wh.createInitContainerBase(pod, *dk)

		require.NotNil(t, initContainer)

		require.NotEmpty(t, initContainer.Args)
		assert.Equal(t, bootstrapper.Use, initContainer.Args[0])
		assert.Equal(t, dtwebhook.InstallContainerName, initContainer.Name)
		assert.Equal(t, initContainer.Image, wh.webhookPodImage)
		assert.NotEmpty(t, initContainer.Resources)

		require.NotNil(t, initContainer.SecurityContext.AllowPrivilegeEscalation)
		assert.False(t, *initContainer.SecurityContext.AllowPrivilegeEscalation)

		require.NotNil(t, initContainer.SecurityContext.Privileged)
		assert.False(t, *initContainer.SecurityContext.Privileged)

		require.NotNil(t, initContainer.SecurityContext.ReadOnlyRootFilesystem)
		assert.True(t, *initContainer.SecurityContext.ReadOnlyRootFilesystem)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, oacommon.DefaultUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, oacommon.DefaultGroup, *initContainer.SecurityContext.RunAsGroup)

		assert.NotNil(t, initContainer.SecurityContext.SeccompProfile)
	})
	t.Run("take security context from user container", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		testUser := ptr.To(int64(420))
		pod.Spec.Containers[0].SecurityContext.RunAsUser = testUser
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = testUser

		initContainer := wh.createInitContainerBase(pod, *dk)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, *testUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *testUser, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("PodSecurityContext overrules defaults", func(t *testing.T) {
		dk := getTestDynakube()
		testUser := ptr.To(int64(420))
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext = nil
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
		pod.Spec.SecurityContext.RunAsUser = testUser
		pod.Spec.SecurityContext.RunAsGroup = testUser

		initContainer := wh.createInitContainerBase(pod, *dk)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, *testUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *testUser, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("should set RunAsNonRoot if root user is used", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext = nil
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
		pod.Spec.SecurityContext.RunAsUser = ptr.To(RootUser)
		pod.Spec.SecurityContext.RunAsGroup = ptr.To(RootGroup)

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.False(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, RootUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, RootGroup, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("should set seccomp profile if feature flag is enabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionSeccompKey: "true"} //nolint:staticcheck
		pod := getTestPod()
		pod.Annotations = map[string]string{}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, initContainer.SecurityContext.SeccompProfile.Type)
	})

	t.Run("should not set suppress-error arg - according to dk", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "fail"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.NotContains(t, initContainer.Args, "--"+k8sinit.SuppressErrorsFlag)
	})

	t.Run("should not set suppress-error arg - according to pod", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "silent"}
		pod := getTestPod()
		pod.Annotations = map[string]string{dtwebhook.AnnotationFailurePolicy: "fail"}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.NotContains(t, initContainer.Args, "--"+k8sinit.SuppressErrorsFlag)
	})

	t.Run("should set suppress-error arg - default", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{}
		pod := getTestPod()
		pod.Annotations = map[string]string{}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.Contains(t, initContainer.Args, "--"+k8sinit.SuppressErrorsFlag)
	})

	t.Run("should set suppress-error arg - unknown value", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "asd"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.Contains(t, initContainer.Args, "--"+k8sinit.SuppressErrorsFlag)

		dk = getTestDynakube()
		dk.Annotations = map[string]string{}
		pod = getTestPod()
		pod.Annotations = map[string]string{dtwebhook.AnnotationFailurePolicy: "asd"}

		initContainer = wh.createInitContainerBase(pod, *dk)

		assert.Contains(t, initContainer.Args, "--"+k8sinit.SuppressErrorsFlag)
	})
}

func createTestHandler(oaMut, metaMut dtwebhook.Mutator, objects ...client.Object) *Handler {
	fakeClient := fake.NewClient(objects...)

	handler := New(
		fakeClient,
		fakeClient,
		events.NewRecorder(record.NewFakeRecorder(10)),
		testWebhookImage,
		false,
		metaMut,
		oaMut,
	)

	return handler
}

func TestAddInitContainerToPod(t *testing.T) {
	t.Run("adds common volumes/mounts", func(t *testing.T) {
		pod := corev1.Pod{}
		initContainer := corev1.Container{}

		addInitContainerToPod(&pod, &initContainer)

		assert.Contains(t, pod.Spec.InitContainers, initContainer)
		require.Len(t, pod.Spec.Volumes, 2)
		assert.True(t, k8svolume.Contains(pod.Spec.Volumes, volumes.ConfigVolumeName))
		assert.True(t, k8svolume.Contains(pod.Spec.Volumes, volumes.InputVolumeName))
		require.Len(t, initContainer.VolumeMounts, 2)
		assert.True(t, k8smount.ContainsPath(initContainer.VolumeMounts, volumes.InitConfigMountPath))
		assert.True(t, k8smount.ContainsPath(initContainer.VolumeMounts, volumes.InitInputMountPath))
	})
}

func Test_combineSecurityContexts(t *testing.T) {
	type testCase struct {
		title            string
		podSc            corev1.PodSecurityContext
		firstContainerSc corev1.SecurityContext
		initContainerSc  corev1.SecurityContext
		expectedOut      corev1.SecurityContext
	}

	cases := []testCase{
		{
			title:            "root pod user",
			podSc:            corev1.PodSecurityContext{RunAsUser: ptr.To(int64(0))},
			firstContainerSc: corev1.SecurityContext{},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsUser: ptr.To(int64(0)), RunAsNonRoot: ptr.To(false)},
		},
		{
			title:            "root pod group",
			podSc:            corev1.PodSecurityContext{RunAsGroup: ptr.To(int64(0))},
			firstContainerSc: corev1.SecurityContext{},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsGroup: ptr.To(int64(0)), RunAsNonRoot: ptr.To(false)},
		},
		{
			title:            "non-root pod user",
			podSc:            corev1.PodSecurityContext{RunAsUser: ptr.To(int64(10))},
			firstContainerSc: corev1.SecurityContext{},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsUser: ptr.To(int64(10)), RunAsNonRoot: ptr.To(true)},
		},
		{
			title:            "non-root pod group",
			podSc:            corev1.PodSecurityContext{RunAsGroup: ptr.To(int64(10))},
			firstContainerSc: corev1.SecurityContext{},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsGroup: ptr.To(int64(10)), RunAsNonRoot: ptr.To(true)},
		},
		{
			title:            "default",
			podSc:            corev1.PodSecurityContext{},
			firstContainerSc: corev1.SecurityContext{},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsNonRoot: ptr.To(true)},
		},
		{
			title:            "non-root user + root group",
			podSc:            corev1.PodSecurityContext{},
			firstContainerSc: corev1.SecurityContext{RunAsUser: ptr.To(int64(10)), RunAsGroup: ptr.To(int64(0))},
			initContainerSc:  corev1.SecurityContext{RunAsUser: ptr.To(int64(55)), RunAsGroup: ptr.To(int64(55))},
			expectedOut:      corev1.SecurityContext{RunAsUser: ptr.To(int64(10)), RunAsGroup: ptr.To(int64(0)), RunAsNonRoot: ptr.To(true)}, // user takes precedence
		},
		{
			title:            "root first container user",
			podSc:            corev1.PodSecurityContext{},
			firstContainerSc: corev1.SecurityContext{RunAsUser: ptr.To(int64(0))},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsUser: ptr.To(int64(0)), RunAsNonRoot: ptr.To(false)},
		},
		{
			title:            "root first container group",
			podSc:            corev1.PodSecurityContext{},
			firstContainerSc: corev1.SecurityContext{RunAsGroup: ptr.To(int64(0))},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsGroup: ptr.To(int64(0)), RunAsNonRoot: ptr.To(false)},
		},
		{
			title:            "non-root first container user",
			podSc:            corev1.PodSecurityContext{},
			firstContainerSc: corev1.SecurityContext{RunAsUser: ptr.To(int64(10))},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsUser: ptr.To(int64(10)), RunAsNonRoot: ptr.To(true)},
		},
		{
			title:            "non-root first container group",
			podSc:            corev1.PodSecurityContext{},
			firstContainerSc: corev1.SecurityContext{RunAsGroup: ptr.To(int64(10))},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsGroup: ptr.To(int64(10)), RunAsNonRoot: ptr.To(true)},
		},
		{
			title:            "first-container overrules pod",
			podSc:            corev1.PodSecurityContext{RunAsUser: ptr.To(int64(10))},
			firstContainerSc: corev1.SecurityContext{RunAsUser: ptr.To(int64(0))},
			initContainerSc:  corev1.SecurityContext{},
			expectedOut:      corev1.SecurityContext{RunAsUser: ptr.To(int64(0)), RunAsNonRoot: ptr.To(false)},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			pod := corev1.Pod{}
			pod.Spec.SecurityContext = &c.podSc
			pod.Spec.Containers = []corev1.Container{
				{
					Name:            "test",
					SecurityContext: &c.firstContainerSc,
				},
			}

			out := combineSecurityContexts(c.initContainerSc, pod)
			require.NotNil(t, out)

			assert.Equal(t, c.expectedOut, *out)
		})
	}
}

func Test_securityContextForInitContainer(t *testing.T) {
	type testCase struct {
		title       string
		dk          dynakube.DynaKube
		isOpenShift bool
		podSc       corev1.PodSecurityContext
		expectedOut corev1.SecurityContext
	}

	cases := []testCase{
		{
			title:       "root pod user",
			dk:          dynakube.DynaKube{},
			isOpenShift: false,
			podSc:       corev1.PodSecurityContext{RunAsUser: ptr.To(int64(0))},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:    ptr.To(int64(0)),
				RunAsGroup:   ptr.To(oacommon.DefaultGroup),
				RunAsNonRoot: ptr.To(false),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
		{
			title:       "root pod group",
			dk:          dynakube.DynaKube{},
			isOpenShift: false,
			podSc:       corev1.PodSecurityContext{RunAsGroup: ptr.To(int64(0))},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:    ptr.To(oacommon.DefaultGroup), // user takes precedence
				RunAsGroup:   ptr.To(int64(0)),
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
		{
			title: "non-root pod user",
			podSc: corev1.PodSecurityContext{RunAsUser: ptr.To(int64(10))},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:    ptr.To(int64(10)),
				RunAsGroup:   ptr.To(oacommon.DefaultGroup),
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
		{
			title:       "non-root pod group",
			dk:          dynakube.DynaKube{},
			isOpenShift: false,
			podSc:       corev1.PodSecurityContext{RunAsGroup: ptr.To(int64(10))},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:    ptr.To(oacommon.DefaultGroup),
				RunAsGroup:   ptr.To(int64(10)),
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
		{
			title:       "default",
			dk:          dynakube.DynaKube{},
			isOpenShift: false,
			podSc:       corev1.PodSecurityContext{},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:    ptr.To(oacommon.DefaultGroup),
				RunAsGroup:   ptr.To(oacommon.DefaultGroup),
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
		{
			title: "non-root user + root group", // does this even make sense?
			podSc: corev1.PodSecurityContext{RunAsUser: ptr.To(int64(10)), RunAsGroup: ptr.To(int64(0))},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:    ptr.To(int64(10)), // user takes precedence
				RunAsGroup:   ptr.To(int64(0)),
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
		{
			title:       "ocp case",
			dk:          dynakube.DynaKube{},
			isOpenShift: true,
			podSc:       corev1.PodSecurityContext{},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsGroup:   ptr.To(oacommon.DefaultGroup),
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
		{
			title: "init seccomp ff set to true",
			dk: dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{exp.InjectionSeccompKey: "true"}}, //nolint:staticcheck
			},
			isOpenShift: true,
			podSc:       corev1.PodSecurityContext{},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsGroup:     ptr.To(oacommon.DefaultGroup),
				RunAsNonRoot:   ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
		},
		{
			title: "init seccomp ff set to false",
			dk: dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{exp.InjectionSeccompKey: "false"}}, //nolint:staticcheck
			},
			isOpenShift: true,
			podSc:       corev1.PodSecurityContext{},
			expectedOut: corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsGroup:   ptr.To(oacommon.DefaultGroup),
				RunAsNonRoot: ptr.To(true),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			pod := corev1.Pod{}
			pod.Spec.SecurityContext = &c.podSc

			out := securityContextForInitContainer(&pod, c.dk, c.isOpenShift)
			require.NotNil(t, out)

			assert.Equal(t, c.expectedOut, *out)
		})
	}
}

func Test_isNonRoot(t *testing.T) {
	t.Run("root user", func(t *testing.T) {
		sc := &corev1.SecurityContext{
			RunAsUser:  ptr.To(int64(0)),
			RunAsGroup: ptr.To(int64(0)),
		}
		assert.False(t, isNonRoot(sc))
	})
	t.Run("non-root user", func(t *testing.T) {
		sc := &corev1.SecurityContext{
			RunAsUser:  ptr.To(int64(1000)),
			RunAsGroup: ptr.To(int64(1000)),
		}
		assert.True(t, isNonRoot(sc))
	})
	t.Run("root user and nil group (OCP case)", func(t *testing.T) {
		sc := &corev1.SecurityContext{
			RunAsUser:  ptr.To(int64(0)),
			RunAsGroup: nil,
		}
		assert.False(t, isNonRoot(sc))
	})
	t.Run("root user and non-root group", func(t *testing.T) {
		sc := &corev1.SecurityContext{
			RunAsUser:  ptr.To(int64(0)),
			RunAsGroup: ptr.To(int64(1000)),
		}
		assert.False(t, isNonRoot(sc))
	})
	t.Run("non-root user and nil group (OCP case)", func(t *testing.T) {
		sc := &corev1.SecurityContext{
			RunAsUser:  ptr.To(int64(1000)),
			RunAsGroup: nil,
		}
		assert.True(t, isNonRoot(sc))
	})
	t.Run("nil user and root group (edge case)", func(t *testing.T) {
		sc := &corev1.SecurityContext{
			RunAsUser:  nil,
			RunAsGroup: ptr.To(int64(0)),
		}
		assert.False(t, isNonRoot(sc))
	})
	t.Run("nil user and non-root group (edge case)", func(t *testing.T) {
		sc := &corev1.SecurityContext{
			RunAsUser:  nil,
			RunAsGroup: ptr.To(int64(1000)),
		}
		assert.True(t, isNonRoot(sc))
	})
	t.Run("nil context", func(t *testing.T) {
		assert.True(t, isNonRoot(nil))
	})
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: getCloudNativeSpec(),
		},
	}
}

func getTestDynakubeMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      testDynakubeName,
		Namespace: testNamespaceName,
	}
}

func getCloudNativeSpec() oneagent.Spec {
	return oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{},
		},
	}
}

func getTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespaceName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "container",
					Image:           "alpine",
					SecurityContext: getTestSecurityContext(),
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "init-container",
					Image: "alpine",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func getTestSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsUser:  ptr.To(testUser),
		RunAsGroup: ptr.To(testUser),
	}
}
