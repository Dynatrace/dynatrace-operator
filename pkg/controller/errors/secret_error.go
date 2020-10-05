package errors

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// handleSecretError is called if getTokenSecret returns an error, or if it returns a nil value as a secret.
func HandleSecretError(secret *corev1.Secret, err error, log logr.Logger) (reconcile.Result, error) {
	if log == nil {
		log = logf.Log.WithName("HandleSecretError")
	}
	if err != nil {
		log.Error(err, err.Error())
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	} else if secret == nil {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, fmt.Errorf("cannot handle 'nil' error and non 'nil' secret ")
}
