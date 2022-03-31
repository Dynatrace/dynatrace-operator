package status

import (
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultUpdateInterval = 5 * time.Minute
)

// The purpose of this struct is to keep track of the state of the Dynakube during a Reconcile.
// Because the dynakube_controller is the one that calls most of the other reconcilers,
// so we pass the to-be-reconciled Dynakube all over the place AND in some cases the reconcilers will update said Dynakube
// therefore we need to keep track of the state for this Dynakube over the whole Reconcile. (was it updated, etc.)
type DynakubeState struct {
	Instance *dynatracev1beta1.DynaKube
	Now      metav1.Time

	// If update is true, then changes on instance will be sent to the Kubernetes API.
	//
	// Additionally, if err is not nil, then the Reconciliation will fail with its value. Unless it's a Too Many
	// Requests HTTP error from the Dynatrace API, on which case, a reconciliation is requeued after one minute delay.
	//
	// If err is nil, then a reconciliation is requeued after requeueAfter.
	Err          error
	Updated      bool
	ValidTokens  bool
	RequeueAfter time.Duration
}

func NewDynakubeState(dk *dynatracev1beta1.DynaKube) *DynakubeState {
	return &DynakubeState{
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

func (dkState *DynakubeState) Update(upd bool, cause string) bool {
	if !upd {
		return false
	}
	log.Info("updating DynaKube CR", "cause", cause, "dynakube", dkState.Instance.Name)
	dkState.Updated = true
	dkState.RequeueAfter = DefaultUpdateInterval
	return true
}

func (dkState *DynakubeState) IsOutdated(last *metav1.Time, threshold time.Duration) bool {
	return last == nil || last.Add(threshold).Before(dkState.Now.Time)
}
