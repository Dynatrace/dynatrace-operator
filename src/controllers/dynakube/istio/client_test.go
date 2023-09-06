package istio

import (
	"context"
	"errors"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clienttest "k8s.io/client-go/testing"
)

func NewTestingClient(fakeClient *fakeistio.Clientset, namespace string) *Client {
	return &Client{
		istioClient: fakeClient,
		namespace:   namespace,
		scheme:      scheme.Scheme,
	}
}

func boomReaction(action clienttest.Action) (handled bool, ret runtime.Object, err error) {
	return true, nil, errors.New("boom")
}

func TestGetVirtualService(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		expectedVirtualService := createTestEmptyVirtualService()
		fakeClient := fakeistio.NewSimpleClientset(expectedVirtualService)
		client := NewTestingClient(fakeClient, expectedVirtualService.Namespace)

		virtualService, err := client.GetVirtualService(ctx, expectedVirtualService.Name)

		require.NoError(t, err)
		assert.Equal(t, expectedVirtualService, virtualService)
	})
	t.Run("does not exist => no error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		testVirtualService := createTestEmptyVirtualService()
		client := NewTestingClient(fakeClient, testVirtualService.Namespace)

		virtualService, err := client.GetVirtualService(ctx, testVirtualService.Name)

		require.NoError(t, err)
		assert.Nil(t, virtualService)
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, "something")

		virtualService, err := client.GetVirtualService(ctx, "random")

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)
		assert.Nil(t, virtualService)
	})
}

func TestCreateVirtualService(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		owner := createTestOwner()
		expectedVirtualService := createTestEmptyVirtualService()
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, expectedVirtualService.Namespace)

		err := client.createVirtualService(ctx, owner, expectedVirtualService)

		require.NoError(t, err)
		serviceEntry, err := fakeClient.NetworkingV1alpha3().VirtualServices(expectedVirtualService.Namespace).Get(ctx, expectedVirtualService.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, expectedVirtualService.Name, serviceEntry.Name)
		assert.Equal(t, expectedVirtualService.Namespace, serviceEntry.Namespace)
		require.NotEmpty(t, expectedVirtualService.OwnerReferences)
		assert.Equal(t, owner.GetName(), expectedVirtualService.OwnerReferences[0].Name)
	})
	t.Run("already exists => error", func(t *testing.T) {
		owner := createTestOwner()
		expectedVirtualService := createTestEmptyVirtualService()
		fakeClient := fakeistio.NewSimpleClientset(expectedVirtualService)
		client := NewTestingClient(fakeClient, expectedVirtualService.Namespace)

		err := client.createVirtualService(ctx, owner, expectedVirtualService)

		require.Error(t, err)
		require.True(t, k8serrors.IsAlreadyExists(err))
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		owner := createTestOwner()
		expectedVirtualService := createTestEmptyVirtualService()
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, expectedVirtualService.Namespace)

		err := client.createVirtualService(ctx, owner, expectedVirtualService)

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)
	})

	t.Run("nil => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, "something")

		err := client.createVirtualService(ctx, nil, nil)

		require.Error(t, err)
	})
}

func TestUpdateVirtualService(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		expectedResourceVersion := "1.2.3"
		oldVirtualService := createTestEmptyVirtualService()
		oldVirtualService.ResourceVersion = expectedResourceVersion
		fakeClient := fakeistio.NewSimpleClientset(oldVirtualService)
		client := NewTestingClient(fakeClient, oldVirtualService.Namespace)

		newVirtualService := createTestEmptyVirtualService()
		addedLabels := map[string]string{
			"test": "test",
		}
		newVirtualService.Labels = addedLabels
		err := client.updateVirtualService(ctx, oldVirtualService, newVirtualService)

		require.NoError(t, err)
		updatedServiceEntry, err := fakeClient.NetworkingV1alpha3().VirtualServices(oldVirtualService.Namespace).Get(ctx, oldVirtualService.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, newVirtualService.Name, updatedServiceEntry.Name)
		assert.Equal(t, newVirtualService.Namespace, updatedServiceEntry.Namespace)
		assert.Equal(t, addedLabels, updatedServiceEntry.Labels)
		assert.Equal(t, expectedResourceVersion, updatedServiceEntry.ResourceVersion)
	})
	t.Run("doesn't exist => return error", func(t *testing.T) {
		newVirtualService := createTestEmptyVirtualService()
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, newVirtualService.Namespace)

		err := client.updateVirtualService(ctx, newVirtualService, newVirtualService)

		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		expectedResourceVersion := "1.2.3"
		oldVirtualService := createTestEmptyVirtualService()
		oldVirtualService.ResourceVersion = expectedResourceVersion
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, oldVirtualService.Namespace)

		newVirtualService := createTestEmptyVirtualService()
		addedLabels := map[string]string{
			"test": "test",
		}
		newVirtualService.Labels = addedLabels
		err := client.updateVirtualService(ctx, oldVirtualService, newVirtualService)

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)

	})
	t.Run("nil => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, "something")

		err := client.updateVirtualService(ctx, nil, nil)

		require.Error(t, err)
	})
}

func TestApplyVirtualService(t *testing.T) {
	ctx := context.Background()
	t.Run("create", func(t *testing.T) {
		owner := createTestOwner()
		expectedVirtualService := createTestEmptyVirtualService()
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, expectedVirtualService.Namespace)

		err := client.ApplyVirtualService(ctx, owner, expectedVirtualService)

		require.NoError(t, err)
		// Get, Create
		assert.Len(t, fakeClient.Actions(), 2)
		virtualService, err := fakeClient.NetworkingV1alpha3().VirtualServices(expectedVirtualService.Namespace).Get(ctx, expectedVirtualService.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, expectedVirtualService.Name, virtualService.Name)
		assert.Equal(t, expectedVirtualService.Namespace, virtualService.Namespace)
		assert.NotEmpty(t, virtualService.Annotations[kubeobjects.AnnotationHash])
		require.NotEmpty(t, virtualService.OwnerReferences)
		assert.Equal(t, owner.GetName(), expectedVirtualService.OwnerReferences[0].Name)
	})
	t.Run("update", func(t *testing.T) {
		owner := createTestOwner()
		expectedResourceVersion := "1.2.3"
		oldVirtualService := createTestEmptyVirtualService()
		oldVirtualService.ResourceVersion = expectedResourceVersion
		fakeClient := fakeistio.NewSimpleClientset(oldVirtualService)
		client := NewTestingClient(fakeClient, oldVirtualService.Namespace)

		newVirtualService := createTestEmptyVirtualService()
		addedLabels := map[string]string{
			"test": "test",
		}
		newVirtualService.Labels = addedLabels
		err := client.ApplyVirtualService(ctx, owner, newVirtualService)

		require.NoError(t, err)
		// Get, Update
		assert.Len(t, fakeClient.Actions(), 2)
		updatedVirtualService, err := fakeClient.NetworkingV1alpha3().VirtualServices(oldVirtualService.Namespace).Get(ctx, oldVirtualService.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, newVirtualService.Name, updatedVirtualService.Name)
		assert.Equal(t, newVirtualService.Namespace, updatedVirtualService.Namespace)
		assert.NotEmpty(t, updatedVirtualService.Annotations[kubeobjects.AnnotationHash])
		assert.Equal(t, addedLabels, updatedVirtualService.Labels)
		assert.Equal(t, expectedResourceVersion, updatedVirtualService.ResourceVersion)
	})
	t.Run("no-change => no update", func(t *testing.T) {
		owner := createTestOwner()
		oldVirtualService := createTestEmptyVirtualService()
		newVirtualService := oldVirtualService.DeepCopy()
		err := kubeobjects.AddHashAnnotation(oldVirtualService)
		require.NoError(t, err)

		fakeClient := fakeistio.NewSimpleClientset(oldVirtualService)
		client := NewTestingClient(fakeClient, oldVirtualService.Namespace)

		err = client.ApplyVirtualService(ctx, owner, newVirtualService)
		require.NoError(t, err)
		// Get
		assert.Len(t, fakeClient.Actions(), 1)
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		owner := createTestOwner()
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, owner.GetNamespace())
		newVirtualService := createTestEmptyVirtualService()

		err := client.ApplyVirtualService(ctx, owner, newVirtualService)

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)

	})
	t.Run("nil => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, "something")

		err := client.ApplyVirtualService(ctx, nil, nil)

		require.Error(t, err)
	})
}


func TestDeleteVirtualService(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		virtualService := createTestEmptyVirtualService()
		fakeClient := fakeistio.NewSimpleClientset(virtualService)
		client := NewTestingClient(fakeClient, virtualService.Namespace)

		err := client.DeleteVirtualService(ctx, virtualService.Name)

		require.NoError(t, err)
		_, err = fakeClient.NetworkingV1alpha3().VirtualServices(virtualService.Namespace).Get(ctx, virtualService.Name, metav1.GetOptions{})
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("does not exist => no error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, "something")

		err := client.DeleteVirtualService(ctx, "random")

		require.NoError(t, err)
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, "something")

		err := client.DeleteVirtualService(ctx, "random")

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)
	})
}

func TestGetServiceEntry(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		expectedServiceEntry := createTestEmptyServiceEntry()
		fakeClient := fakeistio.NewSimpleClientset(expectedServiceEntry)
		client := NewTestingClient(fakeClient, expectedServiceEntry.Namespace)

		serviceEntry, err := client.GetServiceEntry(ctx, expectedServiceEntry.Name)

		require.NoError(t, err)
		assert.Equal(t, expectedServiceEntry, serviceEntry)
	})
	t.Run("does not exist => no error", func(t *testing.T) {
		testServiceEntry := createTestEmptyServiceEntry()
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, testServiceEntry.Namespace)

		serviceEntry, err := client.GetServiceEntry(ctx, testServiceEntry.Name)

		require.NoError(t, err)
		assert.Nil(t, serviceEntry)
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, "doesn't")

		serviceEntry, err := client.GetServiceEntry(ctx, "matter")

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)
		assert.Nil(t, serviceEntry)
	})
}

func TestCreateServiceEntry(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		owner := createTestOwner()
		expectedServiceEntry := createTestEmptyServiceEntry()
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, expectedServiceEntry.Namespace)

		err := client.createServiceEntry(ctx, owner, expectedServiceEntry)

		require.NoError(t, err)
		serviceEntry, err := fakeClient.NetworkingV1alpha3().ServiceEntries(expectedServiceEntry.Namespace).Get(ctx, expectedServiceEntry.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, expectedServiceEntry.Name, serviceEntry.Name)
		assert.Equal(t, expectedServiceEntry.Namespace, serviceEntry.Namespace)
		require.NotEmpty(t, expectedServiceEntry.OwnerReferences)
		assert.Equal(t, owner.GetName(), expectedServiceEntry.OwnerReferences[0].Name)
	})
	t.Run("already exists => error", func(t *testing.T) {
		owner := createTestOwner()
		serviceEntry := createTestEmptyServiceEntry()
		fakeClient := fakeistio.NewSimpleClientset(serviceEntry)
		client := NewTestingClient(fakeClient, serviceEntry.Namespace)

		err := client.createServiceEntry(ctx, owner, serviceEntry)

		require.Error(t, err)
		require.True(t, k8serrors.IsAlreadyExists(err))
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		owner := createTestOwner()
		serviceEntry := createTestEmptyServiceEntry()
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, serviceEntry.Namespace)

		err := client.createServiceEntry(ctx, owner, serviceEntry)

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)
	})

	t.Run("nil => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, "something")

		err := client.createServiceEntry(ctx, nil, nil)

		require.Error(t, err)
	})
}

func TestUpdateServiceEntry(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		expectedResourceVersion := "1.2.3"
		oldServiceEntry := createTestEmptyServiceEntry()
		oldServiceEntry.ResourceVersion = expectedResourceVersion
		fakeClient := fakeistio.NewSimpleClientset(oldServiceEntry)
		client := NewTestingClient(fakeClient, oldServiceEntry.Namespace)

		newServiceEntry := createTestEmptyServiceEntry()
		addedLabels := map[string]string{
			"test": "test",
		}
		newServiceEntry.Labels = addedLabels
		err := client.updateServiceEntry(ctx, oldServiceEntry, newServiceEntry)

		require.NoError(t, err)
		updatedServiceEntry, err := fakeClient.NetworkingV1alpha3().ServiceEntries(oldServiceEntry.Namespace).Get(ctx, oldServiceEntry.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, newServiceEntry.Name, updatedServiceEntry.Name)
		assert.Equal(t, newServiceEntry.Namespace, updatedServiceEntry.Namespace)
		assert.Equal(t, addedLabels, updatedServiceEntry.Labels)
		assert.Equal(t, expectedResourceVersion, updatedServiceEntry.ResourceVersion)
	})
	t.Run("doesn't exist => return error", func(t *testing.T) {
		newServiceEntry := createTestEmptyServiceEntry()
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, newServiceEntry.Namespace)

		err := client.updateServiceEntry(ctx, newServiceEntry, newServiceEntry)

		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		expectedResourceVersion := "1.2.3"
		oldServiceEntry := createTestEmptyServiceEntry()
		oldServiceEntry.ResourceVersion = expectedResourceVersion
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, oldServiceEntry.Namespace)

		newServiceEntry := createTestEmptyServiceEntry()
		addedLabels := map[string]string{
			"test": "test",
		}
		newServiceEntry.Labels = addedLabels
		err := client.updateServiceEntry(ctx, oldServiceEntry, newServiceEntry)

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)

	})
	t.Run("nil => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, "something")

		err := client.updateServiceEntry(ctx, nil, nil)

		require.Error(t, err)
	})
}

func TestApplyServiceEntry(t *testing.T) {
	ctx := context.Background()
	t.Run("create", func(t *testing.T) {
		owner := createTestOwner()
		expectedServiceEntry := createTestEmptyServiceEntry()
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, expectedServiceEntry.Namespace)

		err := client.ApplyServiceEntry(ctx, owner, expectedServiceEntry)

		require.NoError(t, err)
		// Get, Create
		assert.Len(t, fakeClient.Actions(), 2)
		serviceEntry, err := fakeClient.NetworkingV1alpha3().ServiceEntries(expectedServiceEntry.Namespace).Get(ctx, expectedServiceEntry.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, expectedServiceEntry.Name, serviceEntry.Name)
		assert.Equal(t, expectedServiceEntry.Namespace, serviceEntry.Namespace)
		assert.NotEmpty(t, serviceEntry.Annotations[kubeobjects.AnnotationHash])
		require.NotEmpty(t, expectedServiceEntry.OwnerReferences)
		assert.Equal(t, owner.GetName(), expectedServiceEntry.OwnerReferences[0].Name)
	})
	t.Run("update", func(t *testing.T) {
		owner := createTestOwner()
		expectedResourceVersion := "1.2.3"
		oldServiceEntry := createTestEmptyServiceEntry()
		oldServiceEntry.ResourceVersion = expectedResourceVersion
		fakeClient := fakeistio.NewSimpleClientset(oldServiceEntry)
		client := NewTestingClient(fakeClient, oldServiceEntry.Namespace)

		newServiceEntry := createTestEmptyServiceEntry()
		addedLabels := map[string]string{
			"test": "test",
		}
		newServiceEntry.Labels = addedLabels
		err := client.ApplyServiceEntry(ctx, owner, newServiceEntry)

		require.NoError(t, err)
		// Get, Update
		assert.Len(t, fakeClient.Actions(), 2)
		updatedServiceEntry, err := fakeClient.NetworkingV1alpha3().ServiceEntries(oldServiceEntry.Namespace).Get(ctx, oldServiceEntry.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, newServiceEntry.Name, updatedServiceEntry.Name)
		assert.Equal(t, newServiceEntry.Namespace, updatedServiceEntry.Namespace)
		assert.NotEmpty(t, updatedServiceEntry.Annotations[kubeobjects.AnnotationHash])
		assert.Equal(t, addedLabels, updatedServiceEntry.Labels)
		assert.Equal(t, expectedResourceVersion, updatedServiceEntry.ResourceVersion)
	})
	t.Run("no-change => no update", func(t *testing.T) {
		owner := createTestOwner()
		oldServiceEntry := createTestEmptyServiceEntry()
		newServiceEntry := oldServiceEntry.DeepCopy()
		err := kubeobjects.AddHashAnnotation(oldServiceEntry)
		require.NoError(t, err)

		fakeClient := fakeistio.NewSimpleClientset(oldServiceEntry)
		client := NewTestingClient(fakeClient, oldServiceEntry.Namespace)

		err = client.ApplyServiceEntry(ctx, owner, newServiceEntry)
		require.NoError(t, err)
		// Get
		assert.Len(t, fakeClient.Actions(), 1)
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		owner := createTestOwner()
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, owner.GetNamespace())
		newServiceEntry := createTestEmptyServiceEntry()

		err := client.ApplyServiceEntry(ctx, owner, newServiceEntry)

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)

	})
	t.Run("nil => error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, "something")

		err := client.ApplyServiceEntry(ctx, nil, nil)

		require.Error(t, err)
	})
}

func TestDeleteServiceEntry(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		serviceEntry := createTestEmptyServiceEntry()
		fakeClient := fakeistio.NewSimpleClientset(serviceEntry)
		client := NewTestingClient(fakeClient, serviceEntry.Namespace)

		err := client.DeleteServiceEntry(ctx, serviceEntry.Name)

		require.NoError(t, err)
		_, err = fakeClient.NetworkingV1alpha3().ServiceEntries(serviceEntry.Namespace).Get(ctx, serviceEntry.Name, metav1.GetOptions{})
		require.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("does not exist => no error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		client := NewTestingClient(fakeClient, "something")

		err := client.DeleteServiceEntry(ctx, "random")

		require.NoError(t, err)
	})
	t.Run("unknown error => return error", func(t *testing.T) {
		fakeClient := fakeistio.NewSimpleClientset()
		fakeClient.PrependReactor("*", "*", boomReaction)
		client := NewTestingClient(fakeClient, "something")

		err := client.DeleteServiceEntry(ctx, "random")

		require.Error(t, err)
		assert.Len(t, fakeClient.Actions(), 1)
	})
}

func createTestEmptyServiceEntry() *istiov1alpha3.ServiceEntry {
	return &istiov1alpha3.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
}

func createTestEmptyVirtualService() *istiov1alpha3.VirtualService {
	return &istiov1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
}

func createTestOwner() metav1.Object {
	return &dynatracev1beta1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind: "DynaKube",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "owner",
			Namespace: "test",
		},
	}
}
