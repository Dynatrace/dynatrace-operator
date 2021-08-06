package controllers

import (
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Reconciliation struct {
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

func NewReconciliation(log logr.Logger, dk *dynatracev1alpha1.DynaKube) *Reconciliation {
	return &Reconciliation{
		Log:          log,
		Instance:     dk,
		RequeueAfter: 30 * time.Minute,
		Now:          metav1.Now(),
	}
}

func (rec *Reconciliation) Error(err error) bool {
	if err == nil {
		return false
	}
	rec.Err = err
	return true
}

func (rec *Reconciliation) Update(upd bool, d time.Duration, cause string) bool {
	if !upd {
		return false
	}
	rec.Log.Info("Updating DynaKube CR", "cause", cause)
	rec.Updated = true
	rec.RequeueAfter = d
	return true
}

func (rec *Reconciliation) IsOutdated(last *metav1.Time, threshold time.Duration) bool {
	return last == nil || last.Add(threshold).Before(rec.Now.Time)
}
