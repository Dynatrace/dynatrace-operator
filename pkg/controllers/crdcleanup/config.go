package crdcleanup

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log = ctrl.Log.WithName("crdcleanup.controller")
)

func SetLogger(logger logr.Logger) {
	log = logger
}
