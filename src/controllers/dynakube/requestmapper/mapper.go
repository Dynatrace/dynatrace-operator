package requestmapper

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/go-logr/logr"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EnqueueDynakubeRequests enqueues Requests by running a transformation function that outputs a
// DynaKube related reconcile.Requests on Namespace related Event. Namespace Events are filtered
// out to avoid duplicated calls of Reconcile() handler related to the same activity (dynakube controller
// adds labels to namespaces so namespace UPDATE events are received from Watches()).
//
// For UpdateEvents which contain both a new and old object, the transformation function is run on new
// object and one Requests is enqueue.
func EnqueueDynakubeRequests(namespaceName string, l *logr.Logger) handler.EventHandler {
	return &enqueueDynakubeRequests{
		namespaceName: namespaceName,
		log:           l,
	}
}

var _ handler.EventHandler = &enqueueDynakubeRequests{}

type enqueueDynakubeRequests struct {
	namespaceName string
	log           *logr.Logger
}

func (e *enqueueDynakubeRequests) Create(_ context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		e.log.Error(nil, "CREATE event received with no metadata", "event", evt)
		return
	}

	labels := evt.Object.GetLabels()
	if labels != nil {
		if dynakubeName, ok := labels[dtwebhook.InjectionInstanceLabel]; ok {
			e.log.Info("CREATE", "namespace", evt.Object.GetName(), "dynakube", dynakubeName)

			e.enqueue(q, dynakubeName)
			return
		}
	}
	e.log.Info("CREATE - req canceled", "namespace", evt.Object.GetName())
}

func (e *enqueueDynakubeRequests) Update(_ context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) { // nolint:revive
	if evt.ObjectOld == nil {
		e.log.Error(nil, "UPDATE event received with no metadataOld", "event", evt)
		return
	}
	if evt.ObjectNew == nil {
		e.log.Error(nil, "UPDATE event received with no metadataNew ", "event", evt)
		return
	}

	if evt.ObjectNew.GetDeletionTimestamp() != nil {
		e.log.Info("UPDATE before DELETE - req canceled", "namespace", evt.ObjectNew.GetName())
		return
	}

	injectionOld := false
	injectionNew := false
	dynakubeName := ""
	labels := evt.ObjectOld.GetLabels()
	if labels != nil {
		if _, ok := labels[dtwebhook.InjectionInstanceLabel]; ok {
			injectionOld = true
		}
	}
	labels = evt.ObjectNew.GetLabels()
	if labels != nil {
		if name, ok := labels[dtwebhook.InjectionInstanceLabel]; ok {
			injectionNew = true
			dynakubeName = name
		}
	}

	updatedViaCommand := false
	annotations := evt.ObjectNew.GetAnnotations()
	if annotations != nil {
		if _, ok := annotations[mapper.UpdatedViaCommandAnnotation]; ok {
			updatedViaCommand = true
		}
	}

	if !injectionOld && injectionNew && !updatedViaCommand {
		e.log.Info("UPDATE by dynakube - req canceled", "namespace", evt.ObjectNew.GetName())
		// e.logUpdateEvent("UPDATE by dynakube - req canceled", evt)
		return
	}

	if !injectionNew {
		e.log.Info("UPDATE no injection - req canceled", "namespace", evt.ObjectNew.GetName())
		// e.logUpdateEvent("UPDATE no injection - req canceled", evt)
		return
	}

	e.log.Info("UPDATE", "namespace", evt.ObjectNew.GetName(), "dynakube", dynakubeName)
	// e.logUpdateEvent("UPDATE", evt)

	e.enqueue(q, dynakubeName)
}

func (e *enqueueDynakubeRequests) Delete(_ context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		e.log.Error(nil, "DELETE event received with no metadata", "event", evt)
		return
	}
	e.log.Info("DELETE - req canceled", "namespace", evt.Object.GetName())
}

func (e *enqueueDynakubeRequests) Generic(_ context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		e.log.Error(nil, "GENERIC event received with no metadata", "event", evt)
		return
	}
	e.log.Info("GENERIC - req canceled", "namespace", evt.Object.GetName())
}

func (e *enqueueDynakubeRequests) enqueue(q workqueue.RateLimitingInterface, dynakubeName string) {
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      dynakubeName,
			Namespace: e.namespaceName,
		},
	}
	q.AddAfter(request, 20*time.Second)
}

/*
func (e *enqueueDynakubeRequests) logUpdateEvent(msg string, evt event.UpdateEvent) {
	e.log.Info(msg, "name", evt.ObjectNew.GetName(),
		"labelsOld", evt.ObjectOld.GetLabels(), "labelsNew", evt.ObjectNew.GetLabels(),
		"annotationsOld", evt.ObjectOld.GetAnnotations(), "annotationsNew", evt.ObjectNew.GetAnnotations(),
		"createTimeOld", evt.ObjectOld.GetCreationTimestamp(), "createTimeNew", evt.ObjectNew.GetCreationTimestamp(),
		"delTimeOld", evt.ObjectOld.GetDeletionTimestamp(), "delTimeNew", evt.ObjectNew.GetDeletionTimestamp())
}
*/
