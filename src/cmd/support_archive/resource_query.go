package support_archive

import (
	"reflect"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type resourceQueryGroup struct {
	resources []schema.GroupVersionKind
	filters   []client.ListOption
}

type resourceQuery struct {
	groupVersionKind schema.GroupVersionKind
	filters          []client.ListOption
}

func getQueries(namespace string) []resourceQuery {
	allQueries := make([]resourceQuery, 0)
	allQueries = append(allQueries, getInjectedNamespaceQueryGroup().getQueries()...)
	allQueries = append(allQueries, getOperatorNamespaceQueryGroup(namespace).getQueries()...)
	allQueries = append(allQueries, getOperatorComponentsQueryGroup(namespace).getQueries()...)
	allQueries = append(allQueries, getDynakubesQueryGroup(namespace).getQueries()...)
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

func getOperatorComponentsQueryGroup(namespace string) resourceQueryGroup {
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
				kubeobjects.AppNameLabel: "dynatrace-operator",
			},
			client.InNamespace(namespace),
		},
	}
}

func getDynakubesQueryGroup(namespace string) resourceQueryGroup {
	return resourceQueryGroup{
		resources: []schema.GroupVersionKind{
			toGroupVersionKind(dynatracev1beta1.GroupVersion, dynatracev1beta1.DynaKube{}),
		},
		filters: []client.ListOption{
			client.InNamespace(namespace),
		},
	}
}

func toGroupVersionKind(groupVersion schema.GroupVersion, resource interface{}) schema.GroupVersionKind {
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
