package services

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/query"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Query struct {
	query.KubeQuery
}

func NewQuery(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logd.Logger) Query {
	return Query{
		query.New(ctx, kubeClient, kubeReader, log),
	}
}

func (query Query) Get(objectKey client.ObjectKey) (corev1.Service, error) {
	var service corev1.Service
	err := query.KubeReader.Get(query.Ctx, objectKey, &service)

	return service, errors.WithStack(err)
}

func (query Query) Create(service corev1.Service) error {
	query.Log.Info("creating service", "name", service.Name, "namespace", service.Namespace)

	return query.create(service)
}

func (query Query) create(service corev1.Service) error {
	return errors.WithStack(query.KubeClient.Create(query.Ctx, &service))
}

type serviceBuilderData = corev1.Service
type serviceBuilderModifier = builder.Modifier[serviceBuilderData]

type serviceOwnerModifier struct {
	owner metav1.Object
}

func (mod serviceOwnerModifier) Enabled() bool {
	return true
}

func (mod serviceOwnerModifier) Modify(service *corev1.Service) error {
	if err := controllerutil.SetControllerReference(mod.owner, service, scheme.Scheme); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type NameModifier struct {
	name string
}

func NewNameModifier(name string) NameModifier {
	return NameModifier{
		name: name,
	}
}

func (mod NameModifier) Enabled() bool {
	return true
}

func (mod NameModifier) Modify(service *corev1.Service) error {
	service.Name = mod.name

	return nil
}

type NamespaceModifier struct {
	namespaceName string
}

func NewNamespaceModifier(namespaceName string) NamespaceModifier {
	return NamespaceModifier{
		namespaceName: namespaceName,
	}
}

func (mod NamespaceModifier) Enabled() bool {
	return true
}

func (mod NamespaceModifier) Modify(service *corev1.Service) error {
	service.Namespace = mod.namespaceName

	return nil
}

func newServiceOwnerModifier(owner metav1.Object) serviceOwnerModifier {
	return serviceOwnerModifier{
		owner: owner,
	}
}

type PortsModifier struct {
	targetPort intstr.IntOrString
	name       string
	protocol   corev1.Protocol
	port       int32
}

func NewPortsModifier(name string, port int32, protocol corev1.Protocol, targetPort intstr.IntOrString) *PortsModifier {
	return &PortsModifier{name: name, port: port, protocol: protocol, targetPort: targetPort}
}

func (mod PortsModifier) Enabled() bool {
	return true
}

func (mod PortsModifier) Modify(service *corev1.Service) error {
	targetIndex := 0
	for index := range service.Spec.Ports {
		if service.Spec.Ports[targetIndex].Name == mod.name {
			targetIndex = index

			break
		}
	}

	if targetIndex == 0 {
		service.Spec.Ports = make([]corev1.ServicePort, 1)
	}

	service.Spec.Ports[targetIndex].Name = mod.name
	service.Spec.Ports[targetIndex].Port = mod.port
	service.Spec.Ports[targetIndex].Protocol = mod.protocol
	service.Spec.Ports[targetIndex].TargetPort = mod.targetPort

	return nil
}

func Create(owner metav1.Object, mods ...serviceBuilderModifier) (*corev1.Service, error) {
	builderOfService := builder.NewBuilder(corev1.Service{})
	service, err := builderOfService.AddModifier(mods...).AddModifier(newServiceOwnerModifier(owner)).Build()

	return &service, err
}

func (query Query) Delete(name, namespace string) error {
	query.Log.Info("removing service", "name", name, "namespace", namespace)

	return query.delete(name, namespace)
}

func (query Query) delete(name, namespace string) error {
	tmp := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}

	err := query.KubeClient.Delete(query.Ctx, tmp)
	if k8serrors.IsNotFound(err) {
		return nil
	}

	return errors.WithStack(err)
}
