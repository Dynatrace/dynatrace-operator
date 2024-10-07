package oneagent

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	versions "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	versionmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testClusterID = "test-cluster-id"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	namespace := "dynatrace"
	dkName := "dynakube"

	t.Run("create DaemonSet in case OneAgent is needed", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID = "test-tenant"
		fakeClient := fake.NewClient(dk)

		reconciler := &Reconciler{
			client:                   fakeClient,
			apiReader:                fakeClient,
			dk:                       dk,
			versionReconciler:        createVersionReconcilerMock(t),
			connectionInfoReconciler: createConnectionInfoReconcilerMock(t),
			tokens:                   createTokens(),
		}

		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		dsActual := &appsv1.DaemonSet{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: dk.OneAgentDaemonsetName(), Namespace: namespace}, dsActual)
		require.NoError(t, err, "failed to get DaemonSet")
		assert.Equal(t, namespace, dsActual.Namespace, "wrong namespace")
		assert.Equal(t, dk.OneAgentDaemonsetName(), dsActual.GetObjectMeta().GetName(), "wrong name")

		assert.NotNil(t, dsActual.Spec.Template.Spec.Affinity)
	})

	t.Run("remove DaemonSet in case OneAgent is not needed + remove condition", func(t *testing.T) {
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace}}
		setDaemonSetCreatedCondition(dk.Conditions())
		fakeClient := fake.NewClient(dk, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dk.OneAgentDaemonsetName(), Namespace: dk.Namespace}})

		reconciler := &Reconciler{
			client:                   fakeClient,
			apiReader:                fakeClient,
			dk:                       dk,
			versionReconciler:        createVersionReconcilerMock(t),
			connectionInfoReconciler: createConnectionInfoReconcilerMock(t),
		}

		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		dsActual := &appsv1.DaemonSet{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: dk.OneAgentDaemonsetName(), Namespace: namespace}, dsActual)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))

		assert.Nil(t, meta.FindStatusCondition(*dk.Conditions(), oaConditionType))
	})

	t.Run("removing DaemonSet is safe even if its missing", func(t *testing.T) {
		dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace}}
		fakeClient := fake.NewClient(dk)

		reconciler := &Reconciler{
			client:                   fakeClient,
			apiReader:                fakeClient,
			dk:                       dk,
			versionReconciler:        createVersionReconcilerMock(t),
			connectionInfoReconciler: createConnectionInfoReconcilerMock(t),
		}

		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)
	})

	t.Run("NoOneAgentCommunicationHostsError => bubble up error", func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL:      "https://ENVIRONMENTID.live.dynatrace.com/api",
				NetworkZone: "test",
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		}

		connectionInfoReconciler := controllermock.NewReconciler(t)
		connectionInfoReconciler.On("Reconcile",
			mock.AnythingOfType("context.backgroundCtx")).Return(oaconnectioninfo.NoOneAgentCommunicationHostsError).Once()

		fakeClient := fake.NewClient()
		reconciler := &Reconciler{
			client:                   fakeClient,
			apiReader:                fakeClient,
			dk:                       &dk,
			connectionInfoReconciler: connectionInfoReconciler,
			versionReconciler:        createVersionReconcilerMock(t),
		}

		err := reconciler.Reconcile(ctx)
		require.ErrorIs(t, err, oaconnectioninfo.NoOneAgentCommunicationHostsError)
	})

	t.Run("version reconcile fail => return immediately and bubble up error", func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		}

		versionReconciler := versionmock.NewReconciler(t)
		versionReconciler.On("ReconcileOneAgent",
			mock.AnythingOfType("context.backgroundCtx"),
			mock.AnythingOfType("*dynakube.DynaKube")).Return(errors.New("BOOM")).Once()

		fakeClient := fake.NewClient()
		reconciler := &Reconciler{
			client:                   fakeClient,
			apiReader:                fakeClient,
			dk:                       &dk,
			connectionInfoReconciler: controllermock.NewReconciler(t),
			versionReconciler:        versionReconciler,
		}

		err := reconciler.Reconcile(ctx)
		require.Error(t, err)
	})
}

func TestReconcileOneAgent_ReconcileOnEmptyEnvironmentAndDNSPolicy(t *testing.T) {
	ctx := context.Background()
	namespace := "dynatrace"
	dkName := "dynakube"

	dkSpec := dynakube.DynaKubeSpec{
		APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
		Tokens: dkName,
		OneAgent: dynakube.OneAgentSpec{
			ClassicFullStack: &dynakube.HostInjectSpec{
				DNSPolicy: corev1.DNSClusterFirstWithHostNet,
				Labels: map[string]string{
					"label_key": "label_value",
				},
				SecCompProfile: "test",
			},
		},
	}

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec:       dkSpec,
	}

	dk.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo.TenantUUID = "test-tenant"
	dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []dynakube.CommunicationHostStatus{
		{
			Protocol: "http",
			Host:     "dummyhost",
			Port:     666,
		},
	}

	fakeClient := fake.NewClient()
	dtClient := dtclientmock.NewClient(t)

	reconciler := &Reconciler{
		client:                   fakeClient,
		apiReader:                fakeClient,
		dk:                       dk,
		connectionInfoReconciler: createConnectionInfoReconcilerMock(t),
		versionReconciler:        createVersionReconcilerMock(t),
		tokens:                   createTokens(),
	}

	err := reconciler.Reconcile(ctx)
	require.NoError(t, err)

	dsActual := &appsv1.DaemonSet{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: dk.OneAgentDaemonsetName(), Namespace: namespace}, dsActual)
	require.NoError(t, err, "failed to get DaemonSet")
	assert.Equal(t, namespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, dk.OneAgentDaemonsetName(), dsActual.GetObjectMeta().GetName(), "wrong name")
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dsActual.Spec.Template.Spec.DNSPolicy, "wrong policy")
	mock.AssertExpectationsForObjects(t, dtClient)

	condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Equal(t, daemonSetCreatedReason, condition.Reason)
}

func TestReconcile_InstancesSet(t *testing.T) {
	const (
		namespace = "dynatrace"
		name      = "dynakube"
	)

	ctx := context.Background()

	base := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: name,
			OneAgent: dynakube.OneAgentSpec{
				ClassicFullStack: &dynakube.HostInjectSpec{},
			},
		},
	}
	base.Status.OneAgent.ConnectionInfoStatus.TenantUUID = "test-tenant"
	base.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []dynakube.CommunicationHostStatus{
		{
			Protocol: "http",
			Host:     "dummyhost",
			Port:     666,
		},
	}

	c := fake.NewClient()
	oldComponentVersion := "1.186.0.0-0"
	hostIP := "1.2.3.4"

	reconciler := &Reconciler{
		client:    c,
		apiReader: c,
	}

	expectedLabels := map[string]string{
		labels.AppNameLabel:      labels.OneAgentComponentLabel,
		labels.AppComponentLabel: "classicfullstack",
		labels.AppCreatedByLabel: name,
		labels.AppVersionLabel:   oldComponentVersion,
		labels.AppManagedByLabel: version.AppName,
	}

	t.Run("Status.OneAgent.Instances set, if autoUpdate is true", func(t *testing.T) {
		dk := base.DeepCopy()
		reconciler.dk = dk
		reconciler.connectionInfoReconciler = createConnectionInfoReconcilerMock(t)
		reconciler.versionReconciler = createVersionReconcilerMock(t)
		reconciler.tokens = createTokens()
		dk.Status.OneAgent.Version = oldComponentVersion
		dsInfo := daemonset.NewClassicFullStack(dk, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-enabled"
		pod.Namespace = namespace
		pod.Labels = expectedLabels
		pod.Spec = ds.Spec.Template.Spec
		pod.Status.HostIP = hostIP
		err = reconciler.client.Create(ctx, pod)

		require.NoError(t, err)

		err = reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.NotNil(t, dk.Status.OneAgent.Instances)
		assert.NotEmpty(t, dk.Status.OneAgent.Instances)
	})
	t.Run("Status.OneAgent.Instances set, if autoUpdate is false", func(t *testing.T) {
		dk := base.DeepCopy()
		autoUpdate := false
		reconciler.dk = dk
		reconciler.connectionInfoReconciler = createConnectionInfoReconcilerMock(t)
		reconciler.versionReconciler = createVersionReconcilerMock(t)
		reconciler.tokens = createTokens()
		dk.Spec.OneAgent.ClassicFullStack.AutoUpdate = autoUpdate
		dk.Status.OneAgent.Version = oldComponentVersion
		dsInfo := daemonset.NewClassicFullStack(dk, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		pod := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}
		pod.Name = "oneagent-update-disabled"
		pod.Namespace = namespace
		pod.Labels = expectedLabels
		pod.Spec = ds.Spec.Template.Spec
		pod.Status.HostIP = hostIP

		err = reconciler.client.Create(ctx, pod)

		require.NoError(t, err)

		err = reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.NotNil(t, dk.Status.OneAgent.Instances)
		assert.NotEmpty(t, dk.Status.OneAgent.Instances)
	})
}

func TestMigrationForDaemonSetWithoutAnnotation(t *testing.T) {
	dkKey := metav1.ObjectMeta{Name: "my-dynakube", Namespace: "my-namespace"}
	ds1 := &appsv1.DaemonSet{ObjectMeta: dkKey}
	r := Reconciler{}

	dk := &dynakube.DynaKube{
		ObjectMeta: dkKey,
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				HostMonitoring: &dynakube.HostInjectSpec{},
			},
		},
	}

	ds2, err := r.buildDesiredDaemonSet(dk)
	require.NoError(t, err)
	assert.NotEmpty(t, ds2.Annotations[hasher.AnnotationHash])

	assert.True(t, hasher.IsAnnotationDifferent(ds1, ds2))
}

func TestHasSpecChanged(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
		mod      func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube)
	}{
		{
			name:     "hurga",
			expected: false,
			mod:      func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {},
		},
		{
			name:     "image present",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				newDynakube.Status.OneAgent.ImageID = "docker.io/dynatrace/oneagent:1.234.345@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
			},
		},
		{
			name:     "image set but no change",
			expected: false,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				imageId := "docker.io/dynatrace/oneagent:1.234.345@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
				oldDynakube.Status.OneAgent.ImageID = imageId
				newDynakube.Status.OneAgent.ImageID = imageId
			},
		},

		{
			name:     "image changed",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Status.OneAgent.ImageID = "registry.access.redhat.com/dynatrace/oneagent:1.233.345@sha256:6ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
				newDynakube.Status.OneAgent.ImageID = "docker.io/dynatrace/oneagent:1.234.345@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
			},
		},

		{
			name:     "argument removed",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=1", "--set-host-property=OperatorVersion=snapshot"}
				newDynakube.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=1"}
			},
		},

		{
			name:     "argument changed",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=1"}
				newDynakube.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=0"}
			},
		},

		{
			name:     "all arguments removed",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Spec.OneAgent.HostMonitoring.Args = []string{"INFRA_ONLY=1"}
			},
		},

		{
			name:     "resources added",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				newDynakube.Spec.OneAgent.HostMonitoring.OneAgentResources = newResourceRequirements()
			},
		},

		{
			name:     "resources removed",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Spec.OneAgent.HostMonitoring.OneAgentResources = newResourceRequirements()
			},
		},

		{
			name:     "resources removed",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Spec.OneAgent.HostMonitoring.OneAgentResources = newResourceRequirements()
			},
		},

		{
			name:     "priority class added",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				newDynakube.Spec.OneAgent.HostMonitoring.PriorityClassName = "class"
			},
		},

		{
			name:     "priority class removed",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Spec.OneAgent.HostMonitoring.PriorityClassName = "class"
			},
		},

		{
			name:     "priority class set but no change",
			expected: false,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Spec.OneAgent.HostMonitoring.PriorityClassName = "class"
				newDynakube.Spec.OneAgent.HostMonitoring.PriorityClassName = "class"
			},
		},

		{
			name:     "priority class changed",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				oldDynakube.Spec.OneAgent.HostMonitoring.PriorityClassName = "some class"
				newDynakube.Spec.OneAgent.HostMonitoring.PriorityClassName = "other class"
			},
		},

		{
			name:     "dns policy added",
			expected: true,
			mod: func(oldDynakube *dynakube.DynaKube, newDynakube *dynakube.DynaKube) {
				newDynakube.Spec.OneAgent.HostMonitoring.DNSPolicy = corev1.DNSClusterFirst
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := Reconciler{}
			key := metav1.ObjectMeta{Name: "my-oneagent", Namespace: "my-namespace"}
			oldInstance := dynakube.DynaKube{
				ObjectMeta: key,
				Spec: dynakube.DynaKubeSpec{
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{},
					},
				},
			}
			newInstance := dynakube.DynaKube{
				ObjectMeta: key,
				Spec: dynakube.DynaKubeSpec{
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{},
					},
				},
			}
			test.mod(&oldInstance, &newInstance)
			ds1, err := r.buildDesiredDaemonSet(&oldInstance)
			require.NoError(t, err)

			ds2, err := r.buildDesiredDaemonSet(&newInstance)
			require.NoError(t, err)

			assert.NotEmpty(t, ds1.Annotations[hasher.AnnotationHash])
			assert.NotEmpty(t, ds2.Annotations[hasher.AnnotationHash])

			assert.Equal(t, test.expected, hasher.IsAnnotationDifferent(ds1, ds2))
		})
	}
}

func TestNewDaemonset_Affinity(t *testing.T) {
	t.Run(`adds correct affinities`, func(t *testing.T) {
		r := Reconciler{}
		dk := newDynaKube()
		ds, err := r.buildDesiredDaemonSet(dk)

		require.NoError(t, err)
		assert.NotNil(t, ds)

		affinity := ds.Spec.Template.Spec.Affinity

		assert.NotContains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{
				{
					Key:      "beta.kubernetes.io/arch",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
				},
				{
					Key:      "beta.kubernetes.io/os",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"linux"},
				},
			},
		})
		assert.Contains(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{
				{
					Key:      "kubernetes.io/arch",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
				},
				{
					Key:      "kubernetes.io/os",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{"linux"},
				},
			},
		})
	})
}

func newResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    parseQuantity("10m"),
			"memory": parseQuantity("100Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    parseQuantity("20m"),
			"memory": parseQuantity("200Mi"),
		},
	}
}

func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)

	return q
}

func newDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				HostMonitoring: &dynakube.HostInjectSpec{},
			},
		},
	}
}

func TestInstanceStatus(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			OneAgent: dynakube.OneAgentSpec{
				HostMonitoring: &dynakube.HostInjectSpec{},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":          "dynatrace-operator",
				"app.kubernetes.io/component":     "oneagent",
				"app.kubernetes.io/created-by":    dkName,
				"app.kubernetes.io/version":       "snapshot",
				"component.dynatrace.com/feature": deploymentmetadata.HostMonitoringDeploymentType,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
		Status: corev1.PodStatus{
			HostIP: "123.123.123.123",
		},
	}

	fakeClient := fake.NewClient(
		dk,
		pod)

	reconciler := &Reconciler{
		client:    fakeClient,
		apiReader: fakeClient,
	}

	err := reconciler.reconcileInstanceStatuses(context.Background(), dk)
	require.NoError(t, err)
	assert.NotEmpty(t, t, dk.Status.OneAgent.Instances)
	instances := dk.Status.OneAgent.Instances

	err = reconciler.reconcileInstanceStatuses(context.Background(), dk)
	require.NoError(t, err)
	assert.Equal(t, instances, dk.Status.OneAgent.Instances)
}

func TestEmptyInstancesWithWrongLabels(t *testing.T) {
	namespace := "dynatrace"
	dkName := "dynakube"

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			OneAgent: dynakube.OneAgentSpec{
				HostMonitoring: &dynakube.HostInjectSpec{},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: namespace,
			Labels: map[string]string{
				"wrongLabel": "dynatrace-operator",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
		Status: corev1.PodStatus{
			HostIP: "123.123.123.123",
		},
	}

	fakeClient := fake.NewClient(
		dk,
		pod)

	reconciler := &Reconciler{
		client:    fakeClient,
		apiReader: fakeClient,
	}

	err := reconciler.reconcileInstanceStatuses(context.Background(), dk)
	require.NoError(t, err)
	assert.Empty(t, dk.Status.OneAgent.Instances)
}

func TestReconcile_OneAgentConfigMap(t *testing.T) {
	const (
		testTenantUUID      = "test-uuid"
		testTenantEndpoints = "test-endpoints"
	)

	ctx := context.Background()
	dk := newDynaKube()
	dk.Status = dynakube.DynaKubeStatus{
		OneAgent: dynakube.OneAgentStatus{
			ConnectionInfoStatus: dynakube.OneAgentConnectionInfoStatus{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: testTenantUUID,
					Endpoints:  testTenantEndpoints,
				},
				CommunicationHosts: []dynakube.CommunicationHostStatus{
					{
						Protocol: "http",
						Host:     "dummyhost",
						Port:     666,
					},
				},
			},
		},
	}

	fakeClient := fake.NewClient(
		dk)

	t.Run(`create OneAgent connection info ConfigMap`, func(t *testing.T) {
		reconciler := Reconciler{
			dk:                       dk,
			client:                   fakeClient,
			apiReader:                fakeClient,
			versionReconciler:        createVersionReconcilerMock(t),
			connectionInfoReconciler: createConnectionInfoReconcilerMock(t),
			tokens:                   createTokens(),
		}

		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneAgentConnectionInfoConfigMapName(), Namespace: dk.Namespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, actual.Data[connectioninfo.TenantUUIDKey])
		assert.Equal(t, testTenantEndpoints, actual.Data[connectioninfo.CommunicationEndpointsKey])
	})
}

func createConnectionInfoReconcilerMock(t *testing.T) controllers.Reconciler {
	connectionInfoReconciler := controllermock.NewReconciler(t)
	connectionInfoReconciler.On("Reconcile",
		mock.AnythingOfType("context.backgroundCtx")).Return(nil).Once()

	return connectionInfoReconciler
}

func createVersionReconcilerMock(t *testing.T) versions.Reconciler {
	versionReconciler := versionmock.NewReconciler(t)
	versionReconciler.On("ReconcileOneAgent",
		mock.AnythingOfType("context.backgroundCtx"),
		mock.AnythingOfType("*dynakube.DynaKube")).Return(nil).Once()

	return versionReconciler
}

func createTokens() token.Tokens {
	return token.Tokens{
		dtclient.ApiToken: &token.Token{Value: "sdfsdf"},
	}
}
