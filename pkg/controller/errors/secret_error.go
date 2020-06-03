package errors

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// handleSecretError is called if getTokenSecret returns an error, or if it returns a nil value as a secret.
// If err is nil, it assumes secret to be nil and handles error accordingly
func HandleSecretError(err error, log logr.Logger) (reconcile.Result, error) {
	if err != nil {
		log.Error(err, err.Error())
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	// Request object not found, could have been deleted after reconcile request.
	// Return and don't requeue
	return reconcile.Result{}, nil
}
