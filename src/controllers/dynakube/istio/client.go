package istio

import (
	"context"

	"github.com/pkg/errors"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"k8s.io/client-go/discovery"
)

// Client - an adapter for the external istioclientset library
type Client struct {
	istioClient istioclientset.Interface
	scheme      *runtime.Scheme
	namespace   string
}

func NewClient(config *rest.Config, scheme *runtime.Scheme, namespace string) (*Client, error) {
	istioClient, err := istioclientset.NewForConfig(config)

	if err != nil {
		log.Info("failed to initialize istio client", "error", err.Error())
		return nil, errors.WithStack(err)
	}
	return &Client{
		istioClient: istioClient,
		scheme:      scheme,
		namespace:   namespace,
	}, nil
}

// TODO: Maybe move whole check here
func (cl *Client) Discovery() discovery.DiscoveryInterface {
	return cl.istioClient.Discovery()
}

func (cl *Client) ApplyVirtualService(ctx context.Context, owner metav1.Object, virtualService *istiov1alpha3.VirtualService) error {
	if err := controllerutil.SetControllerReference(owner, virtualService, cl.scheme); err != nil {
		return errors.WithStack(err)
	}
	_, err := cl.istioClient.NetworkingV1alpha3().VirtualServices(cl.namespace).Create(ctx, virtualService, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		_, err := cl.istioClient.NetworkingV1alpha3().VirtualServices(cl.namespace).Update(ctx, virtualService, metav1.UpdateOptions{})
		if err != nil {
			log.Info("failed to update virtual service", "name", virtualService.GetName(), "error", err.Error())
			return errors.WithStack(err)
		}
	} else if err != nil {
		log.Info("failed to create virtual service", "name", virtualService.GetName(), "error", err.Error())
		return errors.WithStack(err)
	}
	// TODO: Check if this is actually relevant
	// if createdVirtualService == nil {
	// 	return errors.Errorf("could not create virtual service with spec %v", virtualService.Spec.DeepCopy())
	// }
	return nil
}

func (cl *Client) DeleteVirtualService(ctx context.Context, name string) error {
	err := cl.istioClient.NetworkingV1alpha3().
		VirtualServices(cl.namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		log.Info("failed to remove virtual service", "name", name)
		return err
	}
	return nil
}

func (cl *Client) ApplyServiceEntry(ctx context.Context, owner metav1.Object, serviceEntry *istiov1alpha3.ServiceEntry) error {
	if err := controllerutil.SetControllerReference(owner, serviceEntry, cl.scheme); err != nil {
		return errors.WithStack(err)
	}
	_, err := cl.istioClient.NetworkingV1alpha3().ServiceEntries(cl.namespace).Create(ctx, serviceEntry, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		_, err := cl.istioClient.NetworkingV1alpha3().ServiceEntries(cl.namespace).Update(ctx, serviceEntry, metav1.UpdateOptions{})
		if err != nil {
			log.Info("failed to update service entry", "name", serviceEntry.GetName(), "error", err.Error())
			return errors.WithStack(err)
		}
	} else if err != nil {
		log.Info("failed to create service entry", "name", serviceEntry.GetName(), "error", err.Error())
		return errors.WithStack(err)
	}
	// TODO: Check if this is actually relevant
	// if createdServiceEntry == nil {
	// 	return errors.Errorf("could not create service entry with spec %v", serviceEntry.Spec.DeepCopy())
	// }
	return nil
}

func (cl *Client) DeleteServiceEntry(ctx context.Context, name string) error {
	err := cl.istioClient.NetworkingV1alpha3().
		ServiceEntries(cl.namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		log.Info("failed to remove service entry", "name", name)
		return err
	}
	return nil
}
