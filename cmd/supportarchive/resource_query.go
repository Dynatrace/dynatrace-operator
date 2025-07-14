package supportarchive

import (
	"reflect"

	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest" //nolint:revive
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DefaultEventFieldSelector = fields.OneTermEqualSelector("type", corev1.EventTypeWarning)

type resourceQueryGroup struct {
	resources []schema.GroupVersionKind
	filters   []client.ListOption
}

type resourceQuery struct {
	groupVersionKind schema.GroupVersionKind
	filters          []client.ListOption
}

func getQueries(namespace string, appName string) []resourceQuery {
	allQueries := make([]resourceQuery, 0)
	allQueries = append(allQueries, getInjectedNamespaceQueryGroup().getQueries()...)
	allQueries = append(allQueries, getOperatorNamespaceQueryGroup(namespace).getQueries()...)
	allQueries = append(allQueries, getComponentsQueryGroup(namespace, appName, labels.AppNameLabel).getQueries()...)
	allQueries = append(allQueries, getComponentsQueryGroup(namespace, appName, labels.AppManagedByLabel).getQueries()...)
	allQueries = append(allQueries, getCustomResourcesQueryGroup(namespace).getQueries()...)
	allQueries = append(allQueries, getConfigMapQueryGroup(namespace).getQueries()...)
	allQueries = append(allQueries, getEventsQueryGroup(namespace).getQueries()...)

	return allQueries
}

func getInjectedNamespaceQueryGroup() resourceQueryGroup {
	return resourceQueryGroup{
		resources: []schema.GroupVersionKind{
			toGroupVersionKind(corev1.SchemeGroupVersion, corev1.Namespace{}),
		},
		filters: []client.ListOption{
			client.HasLabels{
				webhook.InjectionInstanceLabel,
			},
		},
	}
}

func getOperatorNamespaceQueryGroup(namespace string) resourceQueryGroup {
	return resourceQueryGroup{
		resources: []schema.GroupVersionKind{
			toGroupVersionKind(corev1.SchemeGroupVersion, corev1.Namespace{}),
		},
		filters: []client.ListOption{
			&client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", namespace),
			},
		},
	}
}

func getComponentsQueryGroup(namespace string, appName string, labelKey string) resourceQueryGroup {
	return resourceQueryGroup{
		resources: []schema.GroupVersionKind{
			toGroupVersionKind(appsv1.SchemeGroupVersion, appsv1.Deployment{}),
			toGroupVersionKind(appsv1.SchemeGroupVersion, appsv1.StatefulSet{}),
			toGroupVersionKind(appsv1.SchemeGroupVersion, appsv1.DaemonSet{}),
			toGroupVersionKind(appsv1.SchemeGroupVersion, appsv1.ReplicaSet{}),
			toGroupVersionKind(corev1.SchemeGroupVersion, corev1.Service{}),
			toGroupVersionKind(corev1.SchemeGroupVersion, corev1.Pod{}),
		},
		filters: []client.ListOption{
			client.MatchingLabels{
				labelKey: appName,
			},
			client.InNamespace(namespace),
		},
	}
}

func getCustomResourcesQueryGroup(namespace string) resourceQueryGroup {
	return resourceQueryGroup{
		resources: []schema.GroupVersionKind{
			toGroupVersionKind(latest.GroupVersion, dynakube.DynaKube{}),
			toGroupVersionKind(v1alpha2.GroupVersion, edgeconnect.EdgeConnect{}),
		},
		filters: []client.ListOption{
			client.InNamespace(namespace),
		},
	}
}

func getConfigMapQueryGroup(namespace string) resourceQueryGroup {
	return resourceQueryGroup{
		resources: []schema.GroupVersionKind{
			toGroupVersionKind(corev1.SchemeGroupVersion, corev1.ConfigMap{}),
		},
		filters: []client.ListOption{
			client.InNamespace(namespace),
		},
	}
}

func getEventsQueryGroup(namespace string) resourceQueryGroup {
	return resourceQueryGroup{
		resources: []schema.GroupVersionKind{
			toGroupVersionKind(corev1.SchemeGroupVersion, corev1.Event{}),
		},
		filters: []client.ListOption{
			client.InNamespace(namespace),
			client.Limit(NumEventsFlagValue),
			&client.ListOptions{
				FieldSelector: DefaultEventFieldSelector,
			},
		},
	}
}

func toGroupVersionKind(groupVersion schema.GroupVersion, resource any) schema.GroupVersionKind {
	typ := reflect.TypeOf(resource)
	typ.Name()
	gvk := schema.GroupVersionKind{
		Group:   groupVersion.Group,
		Version: groupVersion.Version,
		Kind:    typ.Name(),
	}

	return gvk
}

func (q resourceQueryGroup) getQueries() []resourceQuery {
	queries := make([]resourceQuery, 0, len(q.resources))

	for _, resource := range q.resources {
		queries = append(queries, resourceQuery{
			groupVersionKind: resource,
			filters:          q.filters,
		})
	}

	return queries
}
