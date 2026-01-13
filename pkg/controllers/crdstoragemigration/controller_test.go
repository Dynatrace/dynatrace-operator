package crdstoragemigration

import (
	"context"
	"testing"

	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testNamespace = "dynatrace"
)

func TestIsWebhookReady(t *testing.T) {
	t.Run("returns false when replicas is nil", func(t *testing.T) {
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: nil,
			},
		}

		ready := isWebhookReady(deployment)
		assert.False(t, ready)
	})

	t.Run("returns false when ready replicas is less than desired", func(t *testing.T) {
		replicas := int32(3)
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 2,
			},
		}

		ready := isWebhookReady(deployment)
		assert.False(t, ready)
	})

	t.Run("returns false when desired replicas is zero", func(t *testing.T) {
		replicas := int32(0)
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 0,
			},
		}

		ready := isWebhookReady(deployment)
		assert.False(t, ready)
	})

	t.Run("returns true when ready replicas equals desired replicas", func(t *testing.T) {
		replicas := int32(3)
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 3,
			},
		}

		ready := isWebhookReady(deployment)
		assert.True(t, ready)
	})

	t.Run("returns true when ready replicas exceeds desired replicas", func(t *testing.T) {
		replicas := int32(3)
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 4,
			},
		}

		ready := isWebhookReady(deployment)
		assert.True(t, ready)
	})
}

func TestCancelMgr(t *testing.T) {
	t.Run("calls cancel function when set", func(t *testing.T) {
		called := false
		cancelFunc := func() {
			called = true
		}

		controller := &Controller{
			cancelMgrFunc: cancelFunc,
		}

		controller.cancelMgr()
		assert.True(t, called)
	})

	t.Run("does not panic when cancel function is nil", func(t *testing.T) {
		controller := &Controller{
			cancelMgrFunc: nil,
		}

		assert.NotPanics(t, func() {
			controller.cancelMgr()
		})
	})
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("returns no error when webhook deployment not found", func(t *testing.T) {
		clt := fake.NewClient()
		cancelCalled := false

		controller := &Controller{
			client:    clt,
			apiReader: clt,
			cancelMgrFunc: func() {
				cancelCalled = true
			},
		}

		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		}

		result, err := controller.Reconcile(ctx, request)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		assert.False(t, cancelCalled, "cancel should not be called when webhook deployment is not found")
	})

	t.Run("returns error when apiReader.Get fails", func(t *testing.T) {
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return errors.New("fake error")
			},
		})

		controller := &Controller{
			client:    clt,
			apiReader: clt,
		}

		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		}

		result, err := controller.Reconcile(ctx, request)

		require.Error(t, err)
		assert.Equal(t, reconcile.Result{}, result)
	})

	t.Run("returns requeue when webhook deployment not ready", func(t *testing.T) {
		replicas := int32(3)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1, // Not ready
			},
		}

		clt := fake.NewClient(deployment)
		cancelCalled := false

		controller := &Controller{
			client:    clt,
			apiReader: clt,
			cancelMgrFunc: func() {
				cancelCalled = true
			},
		}

		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		}

		result, err := controller.Reconcile(ctx, request)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{RequeueAfter: RetryDuration}, result)
		assert.False(t, cancelCalled, "cancel should not be called when webhook is not ready")
	})

	t.Run("performs storage version migration and calls cancel when webhook ready and migration needed", func(t *testing.T) {
		replicas := int32(1)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}

		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: k8scrd.DynaKubeName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{Name: "v1beta4", Storage: false, Served: true},
					{Name: "v1beta5", Storage: false, Served: true},
					{Name: "v1beta6", Storage: true, Served: true},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta4", "v1beta5", "v1beta6"},
			},
		}

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}

		dk := &latest.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dynakube",
				Namespace: testNamespace,
			},
			Spec: latest.DynaKubeSpec{},
		}

		clt := fake.NewClient(deployment, crd, ns, dk)
		cancelCalled := false

		controller := &Controller{
			client:    clt,
			apiReader: clt,
			cancelMgrFunc: func() {
				cancelCalled = true
			},
		}

		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		}

		result, err := controller.Reconcile(ctx, request)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		assert.True(t, cancelCalled, "cancel should be called after successful storage version migration")

		// Verify CRD was updated
		var updatedCRD apiextensionsv1.CustomResourceDefinition
		err = clt.Get(ctx, client.ObjectKey{Name: k8scrd.DynaKubeName}, &updatedCRD)
		require.NoError(t, err)
		assert.Equal(t, []string{"v1beta6"}, updatedCRD.Status.StoredVersions)
	})

	t.Run("calls cancel when storage version migration not needed", func(t *testing.T) {
		replicas := int32(1)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}

		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: k8scrd.DynaKubeName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{Name: "v1beta6", Storage: true, Served: true},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta6"},
			},
		}

		clt := fake.NewClient(deployment, crd)
		cancelCalled := false

		controller := &Controller{
			client:    clt,
			apiReader: clt,
			cancelMgrFunc: func() {
				cancelCalled = true
			},
		}

		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		}

		result, err := controller.Reconcile(ctx, request)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		assert.True(t, cancelCalled, "cancel should be called even when storage version migration not needed")
	})

	t.Run("returns error when storage version migration fails", func(t *testing.T) {
		replicas := int32(1)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}

		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: k8scrd.DynaKubeName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{Name: "v1beta5", Storage: false, Served: true},
					{Name: "v1beta6", Storage: true, Served: true},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta5", "v1beta6"},
			},
		}

		clt := fake.NewClient(deployment, crd)
		cltWithInterceptor := fake.NewClientWithInterceptors(
			interceptor.Funcs{
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					return errors.New("fake error")
				},
			},
		)

		cancelCalled := false

		controller := &Controller{
			client:    cltWithInterceptor,
			apiReader: clt, // Use normal client for reads
			cancelMgrFunc: func() {
				cancelCalled = true
			},
		}

		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		}

		result, err := controller.Reconcile(ctx, request)

		require.Error(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		assert.False(t, cancelCalled, "cancel should not be called when storage version migration fails")
	})
}
