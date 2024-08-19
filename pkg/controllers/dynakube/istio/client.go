package istio

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/pkg/errors"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ClientBuilder func(config *rest.Config, owner metav1.Object) (*Client, error)

// Client - an adapter for the external istioclientset library
type Client struct {
	IstioClientset istioclientset.Interface
	Owner          metav1.Object
}

func NewClient(config *rest.Config, owner metav1.Object) (*Client, error) {
	istioClient, err := istioclientset.NewForConfig(config)
	if err != nil {
		log.Info("failed to initialize istio client", "error", err.Error())

		return nil, errors.WithStack(err)
	}

	if owner == nil {
		return nil, errors.New("can't create istio client for empty owner")
	}

	return &Client{
		IstioClientset: istioClient,
		Owner:          owner,
	}, nil
}

var _ ClientBuilder = NewClient

func (cl *Client) CheckIstioInstalled() (bool, error) {
	_, err := cl.IstioClientset.Discovery().ServerResourcesForGroupVersion(IstioGVR)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	return err == nil, err
}

func (cl *Client) GetVirtualService(ctx context.Context, name string) (*istiov1beta1.VirtualService, error) {
	virtualService, err := cl.IstioClientset.NetworkingV1beta1().VirtualServices(cl.Owner.GetNamespace()).Get(ctx, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil //nolint:nilnil
	} else if err != nil {
		log.Info("failed to get current virtual service", "name", name, "error", err.Error())

		return nil, errors.WithStack(err)
	}

	return virtualService, nil
}

func (cl *Client) CreateOrUpdateVirtualService(ctx context.Context, newVirtualService *istiov1beta1.VirtualService) error {
	if newVirtualService == nil {
		return errors.New("can't create virtual service based on nil object")
	}

	// the owner reference is created before the hash annotation is added
	if err := controllerutil.SetControllerReference(cl.Owner, newVirtualService, scheme.Scheme); err != nil {
		return errors.WithStack(err)
	}

	delete(newVirtualService.Annotations, hasher.AnnotationHash)

	err := hasher.AddAnnotation(newVirtualService)
	if err != nil {
		return errors.WithMessage(err, "failed to generate and hash annotation for virtual service")
	}

	oldVirtualService, err := cl.GetVirtualService(ctx, newVirtualService.Name)
	if err != nil {
		return err
	}

	if oldVirtualService == nil {
		return cl.createVirtualService(ctx, newVirtualService)
	}

	if !hasher.IsAnnotationDifferent(oldVirtualService, newVirtualService) {
		return nil
	}

	return cl.updateVirtualService(ctx, oldVirtualService, newVirtualService)
}

func (cl *Client) createVirtualService(ctx context.Context, virtualService *istiov1beta1.VirtualService) error {
	if virtualService == nil {
		return errors.New("can't create virtual service based on nil object")
	}

	_, err := cl.IstioClientset.NetworkingV1beta1().VirtualServices(cl.Owner.GetNamespace()).Create(ctx, virtualService, metav1.CreateOptions{})
	if err != nil {
		log.Info("failed to create virtual service", "name", virtualService.GetName(), "error", err.Error())

		return errors.WithStack(err)
	}

	return nil
}

func (cl *Client) updateVirtualService(ctx context.Context, oldVirtualService, newVirtualService *istiov1beta1.VirtualService) error {
	if oldVirtualService == nil || newVirtualService == nil {
		return errors.New("can't update service entry based on nil object")
	}

	newVirtualService.ObjectMeta.ResourceVersion = oldVirtualService.ObjectMeta.ResourceVersion
	_, err := cl.IstioClientset.NetworkingV1beta1().VirtualServices(cl.Owner.GetNamespace()).Update(ctx, newVirtualService, metav1.UpdateOptions{})

	if err != nil {
		log.Info("failed to update virtual service", "name", newVirtualService.GetName(), "error", err.Error())

		return errors.WithStack(err)
	}

	return nil
}

func (cl *Client) DeleteVirtualService(ctx context.Context, name string) error {
	err := cl.IstioClientset.NetworkingV1beta1().
		VirtualServices(cl.Owner.GetNamespace()).
		Delete(ctx, name, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		log.Info("failed to remove virtual service", "name", name)

		return errors.WithStack(err)
	}

	return nil
}

func (cl *Client) GetServiceEntry(ctx context.Context, name string) (*istiov1beta1.ServiceEntry, error) {
	serviceEntry, err := cl.IstioClientset.NetworkingV1beta1().ServiceEntries(cl.Owner.GetNamespace()).Get(ctx, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil //nolint:nilnil
	} else if err != nil {
		log.Info("failed to get current service entry", "name", name, "error", err.Error())

		return nil, errors.WithStack(err)
	}

	return serviceEntry, nil
}

func (cl *Client) CreateOrUpdateServiceEntry(ctx context.Context, newServiceEntry *istiov1beta1.ServiceEntry) error {
	if newServiceEntry == nil {
		return errors.New("can't create service entry based on nil object")
	}

	// the owner reference is created before the hash annotation is added
	if err := controllerutil.SetControllerReference(cl.Owner, newServiceEntry, scheme.Scheme); err != nil {
		return errors.WithStack(err)
	}

	delete(newServiceEntry.Annotations, hasher.AnnotationHash)

	err := hasher.AddAnnotation(newServiceEntry)
	if err != nil {
		return errors.WithMessage(err, "failed to generate and hash annotation for service entry")
	}

	oldServiceEntry, err := cl.GetServiceEntry(ctx, newServiceEntry.Name)
	if err != nil {
		return err
	}

	if oldServiceEntry == nil {
		return cl.createServiceEntry(ctx, newServiceEntry)
	}

	if !hasher.IsAnnotationDifferent(oldServiceEntry, newServiceEntry) {
		return nil
	}

	return cl.updateServiceEntry(ctx, oldServiceEntry, newServiceEntry)
}

func (cl *Client) createServiceEntry(ctx context.Context, serviceEntry *istiov1beta1.ServiceEntry) error {
	if serviceEntry == nil {
		return errors.New("can't create service entry based on nil object")
	}

	_, err := cl.IstioClientset.NetworkingV1beta1().ServiceEntries(cl.Owner.GetNamespace()).Create(ctx, serviceEntry, metav1.CreateOptions{})
	if err != nil {
		log.Info("failed to create service entry", "name", serviceEntry.GetName(), "error", err.Error())

		return errors.WithStack(err)
	}

	return nil
}

func (cl *Client) updateServiceEntry(ctx context.Context, oldServiceEntry, newServiceEntry *istiov1beta1.ServiceEntry) error {
	if oldServiceEntry == nil || newServiceEntry == nil {
		return errors.New("can't update service entry based on nil object")
	}

	newServiceEntry.ObjectMeta.ResourceVersion = oldServiceEntry.ObjectMeta.ResourceVersion
	_, err := cl.IstioClientset.NetworkingV1beta1().ServiceEntries(cl.Owner.GetNamespace()).Update(ctx, newServiceEntry, metav1.UpdateOptions{})

	if err != nil {
		log.Info("failed to update service entry", "name", newServiceEntry.GetName(), "error", err.Error())

		return errors.WithStack(err)
	}

	return nil
}

func (cl *Client) DeleteServiceEntry(ctx context.Context, name string) error {
	err := cl.IstioClientset.NetworkingV1beta1().
		ServiceEntries(cl.Owner.GetNamespace()).
		Delete(ctx, name, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		log.Info("failed to remove service entry", "name", name)

		return errors.WithStack(err)
	}

	return nil
}
