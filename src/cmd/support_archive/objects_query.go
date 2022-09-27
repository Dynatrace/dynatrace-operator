package support_archive

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getObjectsQuery(ctx *supportArchiveContext) []objectQuery {
	return []objectQuery{
		{
			groupVersionKind: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "NamespaceList",
			},
		},
		{
			groupVersionKind: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "DeploymentList",
			},
			listOptions: []client.ListOption{
				client.InNamespace(ctx.namespaceName),
			},
		},
		{
			groupVersionKind: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "StatefulSetList",
			},
			listOptions: []client.ListOption{
				client.InNamespace(ctx.namespaceName),
			},
		},
		{
			groupVersionKind: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "DaemonSetList",
			},
			listOptions: []client.ListOption{
				client.InNamespace(ctx.namespaceName),
			},
		},
		{
			groupVersionKind: schema.GroupVersionKind{
				Group:   "dynatrace.com",
				Version: "v1beta1",
				Kind:    "DynaKubeList",
			},
			listOptions: []client.ListOption{
				client.InNamespace(ctx.namespaceName),
			},
		},
		{
			groupVersionKind: schema.GroupVersionKind{
				Group:   "networking.istio.io",
				Version: "v1alpha3",
				Kind:    "VirtualServiceList",
			},
		},
		{
			groupVersionKind: schema.GroupVersionKind{
				Group:   "networking.istio.io",
				Version: "v1alpha3",
				Kind:    "ServiceEntryList",
			},
		},
	}
}
