package authorization

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/app"
	hostvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/host"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testOperatorNamespace = "dynatrace"
	testDynakubeName      = "my-dynakube"
	testPodName           = "oneagent-abc"
	testPodNamespace      = "user-namespace"
	testPodUID            = "pod-uid-1234"
)

func Test_Authorizer_Authorize(t *testing.T) {
	t.Run("app mode", func(t *testing.T) {
		baseNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testPodNamespace,
				Labels: map[string]string{
					dtwebhook.InjectionInstanceLabel: testDynakubeName,
				},
			},
		}

		baseCfg := csivolumes.VolumeConfig{
			PodName:      testPodName,
			PodNamespace: testPodNamespace,
			PodUID:       testPodUID,
			Mode:         appvolumes.Mode,
			DynakubeName: testDynakubeName,
		}

		t.Run("namespace label matches supplied dynakube => authorized", func(t *testing.T) {
			auth := New(fake.NewClient(&baseNamespace), testOperatorNamespace)

			dynakubeName, err := auth.Authorize(t.Context(), baseCfg)

			require.NoError(t, err)
			assert.Equal(t, testDynakubeName, dynakubeName)
		})

		t.Run("namespace missing injection label => denied", func(t *testing.T) {
			ns := baseNamespace.DeepCopy()
			ns.Labels = map[string]string{}
			auth := New(fake.NewClient(ns), testOperatorNamespace)

			dynakubeName, err := auth.Authorize(t.Context(), baseCfg)

			require.ErrorIs(t, err, errAccessDenied)
			assert.Empty(t, dynakubeName)
		})

		t.Run("wrong Dynakube name => denied", func(t *testing.T) {
			auth := New(fake.NewClient(&baseNamespace), testOperatorNamespace)
			cfg := baseCfg
			cfg.DynakubeName = "some-dynakube"

			dynakubeName, err := auth.Authorize(t.Context(), cfg)

			require.ErrorIs(t, err, errAccessDenied)
			assert.Empty(t, dynakubeName)
		})

		t.Run("namespace not found => denied", func(t *testing.T) {
			auth := New(fake.NewClient(), testOperatorNamespace)

			dynakubeName, err := auth.Authorize(t.Context(), baseCfg)

			require.ErrorIs(t, err, errAccessDenied)
			assert.Empty(t, dynakubeName)
		})
	})

	t.Run("host mode", func(t *testing.T) {
		basePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testPodName,
				Namespace: testOperatorNamespace,
				UID:       types.UID(testPodUID),
				Labels: map[string]string{
					k8slabel.AppManagedByLabel: version.AppName,
					k8slabel.AppNameLabel:      k8slabel.OneAgentComponentLabel,
					k8slabel.AppCreatedByLabel: testDynakubeName,
				},
			},
		}

		baseCfg := csivolumes.VolumeConfig{
			PodName:      testPodName,
			PodNamespace: testOperatorNamespace,
			PodUID:       testPodUID,
			Mode:         hostvolumes.Mode,
			DynakubeName: testDynakubeName,
		}

		t.Run("valid OneAgent pod => authorized", func(t *testing.T) {
			auth := New(fake.NewClient(&basePod), testOperatorNamespace)

			dynakubeName, err := auth.Authorize(t.Context(), baseCfg)

			require.NoError(t, err)
			assert.Equal(t, testDynakubeName, dynakubeName)
		})

		t.Run("pod not in operator namespace => denied", func(t *testing.T) {
			auth := New(fake.NewClient(), testOperatorNamespace)
			cfg := baseCfg
			cfg.PodNamespace = "other-namespace"

			dynakubeName, err := auth.Authorize(t.Context(), cfg)

			require.ErrorIs(t, err, errAccessDenied)
			assert.Empty(t, dynakubeName)
		})

		t.Run("pod UID mismatch => denied", func(t *testing.T) {
			pod := basePod.DeepCopy()
			pod.UID = "some-super-random-uid"
			auth := New(fake.NewClient(pod), testOperatorNamespace)

			dynakubeName, err := auth.Authorize(t.Context(), baseCfg)

			require.ErrorIs(t, err, errAccessDenied)
			assert.Empty(t, dynakubeName)
		})

		t.Run("pod has wrong managed-by label => denied", func(t *testing.T) {
			pod := basePod.DeepCopy()
			pod.Labels[k8slabel.AppManagedByLabel] = "super-random-123"
			auth := New(fake.NewClient(pod), testOperatorNamespace)

			dynakubeName, err := auth.Authorize(t.Context(), baseCfg)

			require.ErrorIs(t, err, errAccessDenied)
			assert.Empty(t, dynakubeName)
		})

		t.Run("pod missing name=oneagent label => denied", func(t *testing.T) {
			pod := basePod.DeepCopy()
			pod.Labels[k8slabel.AppNameLabel] = "definitely-not-oneagent"
			auth := New(fake.NewClient(pod), testOperatorNamespace)

			dynakubeName, err := auth.Authorize(t.Context(), baseCfg)

			require.ErrorIs(t, err, errAccessDenied)
			assert.Empty(t, dynakubeName)
		})

		t.Run("wrong Dynakube name => denied", func(t *testing.T) {
			auth := New(fake.NewClient(&basePod), testOperatorNamespace)
			cfg := baseCfg
			cfg.DynakubeName = "other-dynakube"

			dynakubeName, err := auth.Authorize(t.Context(), cfg)

			require.ErrorIs(t, err, errAccessDenied)
			assert.Empty(t, dynakubeName)
		})
	})

	t.Run("unknown mode => denied", func(t *testing.T) {
		auth := New(fake.NewClient(), testOperatorNamespace)
		cfg := csivolumes.VolumeConfig{
			PodName:      testPodName,
			PodNamespace: testPodNamespace,
			PodUID:       testPodUID,
			Mode:         "unknown",
			DynakubeName: testDynakubeName,
		}

		dynakubeName, err := auth.Authorize(t.Context(), cfg)

		require.ErrorIs(t, err, errAccessDenied)
		assert.Empty(t, dynakubeName)
	})
}
