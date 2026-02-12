//go:build e2e

package events

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sevent"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func LogEvents(ctx context.Context, c *envconf.Config, t *testing.T) {
	klog.InfoS("test failed", "f", t.Name(), "failed", t.Failed())

	resource := c.Client().Resources()

	optFunc := func(options *metav1.ListOptions) {
		options.Limit = int64(300)
		options.FieldSelector = fmt.Sprint(fields.OneTermEqualSelector("type", corev1.EventTypeWarning))
	}

	events := k8sevent.List(t, ctx, resource, "dynatrace", optFunc)

	klog.InfoS("Events list", "events total", len(events.Items))
	for _, eventItem := range events.Items {
		klog.InfoS("Event", "name", eventItem.Name, "message", eventItem.Message, "reason", eventItem.Reason, "type", eventItem.Type)
	}
}
