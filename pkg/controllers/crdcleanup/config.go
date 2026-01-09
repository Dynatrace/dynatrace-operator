package crdcleanup

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log = ctrl.Log.WithName("crdcleanup.controller")
)
