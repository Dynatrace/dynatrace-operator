package istio

import (
	"context"

	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	"github.com/pkg/errors"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ClientBuilder func(config *rest.Config, scheme *runtime.Scheme, dynaKube *dynakubev1beta1.DynaKube) (*Client, error)

// Client - an adapter for the external istioclientset library
type Client struct {
	IstioClientset istioclientset.Interface
	Scheme         *runtime.Scheme
	Dynakube       *dynakubev1beta1.DynaKube
}

func NewClient(config *rest.Config, scheme *runtime.Scheme, dynakube *dynakubev1beta1.DynaKube) (*Client, error) {
	istioClient, err := istioclientset.NewForConfig(config)

	if err != nil {
		log.Info("failed to initialize istio client", "error", err.Error())
		return nil, errors.WithStack(err)
	}
	if dynakube == nil {
		return nil, errors.New("can't create istio client for empty dynakube")
	}

	return &Client{
		IstioClientset: istioClient,
		Scheme:         scheme,
		Dynakube:       dynakube,
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

func (cl *Client) GetVirtualService(ctx context.Context, name string) (*istiov1alpha3.VirtualService, error) {
	virtualService, err := cl.IstioClientset.NetworkingV1alpha3().VirtualServices(cl.Dynakube.Namespace).Get(ctx, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		log.Info("failed to get current virtual service", "name", name, "error", err.Error())
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (cl *Client) CreateOrUpdateVirtualService(ctx context.Context, newVirtualService *istiov1alpha3.VirtualService) error {
	if newVirtualService == nil {
		return errors.New("can't create virtual service based on nil object")
	}

	// the owner reference is created before the hash annotation is added
	if err := controllerutil.SetControllerReference(cl.Dynakube, newVirtualService, cl.Scheme); err != nil {
		return errors.WithStack(err)
	}

	delete(newVirtualService.Annotations, kubeobjects.AnnotationHash)
	err := kubeobjects.AddHashAnnotation(newVirtualService)
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

	if !kubeobjects.IsHashAnnotationDifferent(oldVirtualService, newVirtualService) {
		return nil
	}
	return cl.updateVirtualService(ctx, oldVirtualService, newVirtualService)
}

func (cl *Client) createVirtualService(ctx context.Context, virtualService *istiov1alpha3.VirtualService) error {
	if virtualService == nil {
		return errors.New("can't create virtual service based on nil object")
	}
	_, err := cl.IstioClientset.NetworkingV1alpha3().VirtualServices(cl.Dynakube.Namespace).Create(ctx, virtualService, metav1.CreateOptions{})
	if err != nil {
		log.Info("failed to create virtual service", "name", virtualService.GetName(), "error", err.Error())
		return errors.WithStack(err)
	}
	return nil
}

func (cl *Client) updateVirtualService(ctx context.Context, oldVirtualService, newVirtualService *istiov1alpha3.VirtualService) error {
	if oldVirtualService == nil || newVirtualService == nil {
		return errors.New("can't update service entry based on nil object")
	}
	newVirtualService.ObjectMeta.ResourceVersion = oldVirtualService.ObjectMeta.ResourceVersion
	_, err := cl.IstioClientset.NetworkingV1alpha3().VirtualServices(cl.Dynakube.Namespace).Update(ctx, newVirtualService, metav1.UpdateOptions{})
	if err != nil {
		log.Info("failed to update virtual service", "name", newVirtualService.GetName(), "error", err.Error())
		return errors.WithStack(err)
	}
	return nil
}

func (cl *Client) DeleteVirtualService(ctx context.Context, name string) error {
	err := cl.IstioClientset.NetworkingV1alpha3().
		VirtualServices(cl.Dynakube.Namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		log.Info("failed to remove virtual service", "name", name)
		return errors.WithStack(err)
	}
	return nil
}

func (cl *Client) GetServiceEntry(ctx context.Context, name string) (*istiov1alpha3.ServiceEntry, error) {
	serviceEntry, err := cl.IstioClientset.NetworkingV1alpha3().ServiceEntries(cl.Dynakube.Namespace).Get(ctx, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		log.Info("failed to get current service entry", "name", name, "error", err.Error())
		return nil, errors.WithStack(err)
	}
	return serviceEntry, nil
}

func (cl *Client) CreateOrUpdateServiceEntry(ctx context.Context, newServiceEntry *istiov1alpha3.ServiceEntry) error {
	if newServiceEntry == nil {
		return errors.New("can't create service entry based on nil object")
	}

	// the owner reference is created before the hash annotation is added
	if err := controllerutil.SetControllerReference(cl.Dynakube, newServiceEntry, cl.Scheme); err != nil {
		return errors.WithStack(err)
	}

	delete(newServiceEntry.Annotations, kubeobjects.AnnotationHash)
	err := kubeobjects.AddHashAnnotation(newServiceEntry)
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

	if !kubeobjects.IsHashAnnotationDifferent(oldServiceEntry, newServiceEntry) {
		return nil
	}
	return cl.updateServiceEntry(ctx, oldServiceEntry, newServiceEntry)
}

func (cl *Client) createServiceEntry(ctx context.Context, serviceEntry *istiov1alpha3.ServiceEntry) error {
	if serviceEntry == nil {
		return errors.New("can't create service entry based on nil object")
	}

	_, err := cl.IstioClientset.NetworkingV1alpha3().ServiceEntries(cl.Dynakube.Namespace).Create(ctx, serviceEntry, metav1.CreateOptions{})
	if err != nil {
		log.Info("failed to create service entry", "name", serviceEntry.GetName(), "error", err.Error())
		return errors.WithStack(err)
	}
	return nil
}

func (cl *Client) updateServiceEntry(ctx context.Context, oldServiceEntry, newServiceEntry *istiov1alpha3.ServiceEntry) error {
	if oldServiceEntry == nil || newServiceEntry == nil {
		return errors.New("can't update service entry based on nil object")
	}

	newServiceEntry.ObjectMeta.ResourceVersion = oldServiceEntry.ObjectMeta.ResourceVersion
	_, err := cl.IstioClientset.NetworkingV1alpha3().ServiceEntries(cl.Dynakube.Namespace).Update(ctx, newServiceEntry, metav1.UpdateOptions{})
	if err != nil {
		log.Info("failed to update service entry", "name", newServiceEntry.GetName(), "error", err.Error())
		return errors.WithStack(err)
	}
	return nil
}

func (cl *Client) DeleteServiceEntry(ctx context.Context, name string) error {
	err := cl.IstioClientset.NetworkingV1alpha3().
		ServiceEntries(cl.Dynakube.Namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if !k8serrors.IsNotFound(err) {
		log.Info("failed to remove service entry", "name", name)
		return errors.WithStack(err)
	}
	return nil
}
