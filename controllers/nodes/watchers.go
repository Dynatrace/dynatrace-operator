package nodes

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	toolscache "k8s.io/client-go/tools/cache"
)

// watchDeletions returns a channel where Node deletion operations will be notified.
func (r *ReconcileNodes) watchDeletions(stop <-chan struct{}) (chan string, error) {
	ifm, err := r.cache.GetInformer(context.TODO(), &corev1.Node{})
	if err != nil {
		return nil, err
	}

	// Don't close this channel and leak it on purpose to avoid panicking if the Informer sends a notification
	// after we stop, but it's not worth it to have it synchronized.
	chDels := make(chan string, 20)

	ifm.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if o, err := meta.Accessor(obj); err == nil {
				chDels <- o.GetName()
			} else {
				r.logger.Error(err, "missing Meta", "object", obj, "type", fmt.Sprintf("%T", obj))
			}
		},
	})

	return chDels, nil
}

func (r *ReconcileNodes) watchUpdates() (chan string, error) {
	informer, err := r.cache.GetInformer(context.TODO(), &corev1.Node{})
	if err != nil {
		return nil, err
	}

	chUpdates := make(chan string, 20)

	informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		UpdateFunc: r.handleUpdate(chUpdates),
	})

	return chUpdates, nil
}

func (r *ReconcileNodes) handleUpdate(chUpdates chan string) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		newMeta, err := meta.Accessor(newObj)
		if err != nil {
			r.logger.Error(err, "missing Meta",
				"new object", newObj, "type", fmt.Sprintf("%T", newObj))
			return
		}

		chUpdates <- newMeta.GetName()
	}
}

// watchTicks returns a channel where tick messages will be sent periodically.
//
// Unlike time.Ticker, this function will also send an initial tick.
func watchTicks(stop <-chan struct{}, d time.Duration) <-chan struct{} {
	chAll := make(chan struct{}, 1)
	chAll <- struct{}{}

	go func() {
		defer close(chAll)

		ticker := time.NewTicker(d)
		defer ticker.Stop()

		for range ticker.C {
			chAll <- struct{}{}
		}
	}()

	return chAll
}
