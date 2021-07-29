package webhook

import (
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func AddDynakubeValidationWebhookToManager(manager ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(manager).
		For(&dynakubeValidator{}).
		Complete()
}

var _ webhook.Validator = &dynakubeValidator{}

type dynakubeValidator struct {
	logger logr.Logger
}

func newDynakubeValidator() webhook.Validator {
	return &dynakubeValidator{
		logger: logger.NewDTLogger(),
	}
}

func (d *dynakubeValidator) GetObjectKind() schema.ObjectKind {
	panic("implement me")
}

func (d *dynakubeValidator) DeepCopyObject() runtime.Object {
	panic("implement me")
}

func (d *dynakubeValidator) ValidateCreate() error {
	panic("implement me")
}

func (d *dynakubeValidator) ValidateUpdate(old runtime.Object) error {
	panic("implement me")
}

func (d *dynakubeValidator) ValidateDelete() error {
	panic("implement me")
}
