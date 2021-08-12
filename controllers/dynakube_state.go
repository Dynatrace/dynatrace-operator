package controllers

import (
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The purpose of this struct is to keep track of the state of the Dynakube during a Reconcile.
// Because the dynakube_controller is the one that calls most of the other reconcilers,
// so we pass the to-be-reconciled Dynakube all over the place AND in some cases the reconcilers will update said Dynakube
// therefore we need to keep track of the state for this Dynakube over the whole Reconcile. (was it updated, etc.)
type DynakubeState struct {
	Log      logr.Logger
	Instance *dynatracev1alpha1.DynaKube
	Now      metav1.Time

	// If update is true, then changes on instance will be sent to the Kubernetes API.
	//
	// Additionally, if err is not nil, then the Reconciliation will fail with its value. Unless it's a Too Many
	// Requests HTTP error from the Dynatrace API, on which case, a reconciliation is requeued after one minute delay.
	//
	// If err is nil, then a reconciliation is requeued after requeueAfter.
	Err          error
	Updated      bool
	RequeueAfter time.Duration
}

func NewDynakubeState(log logr.Logger, dk *dynatracev1alpha1.DynaKube) *DynakubeState {
	return &DynakubeState{
		Log:          log,
		Instance:     dk,
		RequeueAfter: 30 * time.Minute,
		Now:          metav1.Now(),
	}
}

func (dkState *DynakubeState) Error(err error) bool {
	if err == nil {
		return false
	}
	dkState.Err = err
	return true
}

func (dkState *DynakubeState) Update(upd bool, d time.Duration, cause string) bool {
	if !upd {
		return false
	}
	dkState.Log.Info("Updating DynaKube CR", "cause", cause)
	dkState.Updated = true
	dkState.RequeueAfter = d
	return true
}

func (dkState *DynakubeState) IsOutdated(last *metav1.Time, threshold time.Duration) bool {
	return last == nil || last.Add(threshold).Before(dkState.Now.Time)
}
