package webhook

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func AddDynakubeValidationWebhookToManager(manager ctrl.Manager) error {
	manager.GetWebhookServer().Register("/validate", &webhook.Admission{
		Handler: newDynakubeValidator(),
	})
	manager.GetWebhookServer().Register("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	return nil
}

type dynakubeValidator struct {
	logger logr.Logger
}

func (validator *dynakubeValidator) Handle(_ context.Context, _ admission.Request) admission.Response {
	validator.logger.Info("handling request, yay!")
	return admission.Errored(404, errors.New("not implemented"))
}

func newDynakubeValidator() admission.Handler {
	return &dynakubeValidator{
		logger: logger.NewDTLogger(),
	}
}
