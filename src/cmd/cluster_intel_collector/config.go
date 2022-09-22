package cluster_intel_collector

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = newLogCollectorLogger("[log collector]")
)

type intelCollectorContext struct {
	ctx           context.Context
	clientSet     kubernetes.Interface // used to get access to logs
	apiReader     client.Reader        // used for manifest collection
	namespaceName string               // the default namespace ("dynatrace") or provided in the command line
	toStdout      bool
	targetDir     string
}

type manifestSpec struct {
	gvk         schema.GroupVersionKind
	listOptions []client.ListOption
}

func getRelevantManifests(ctx *intelCollectorContext) []manifestSpec {
	return []manifestSpec{
		{
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "NamespaceList",
			},
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "DeploymentList",
			},
			listOptions: []client.ListOption{
				client.InNamespace(ctx.namespaceName),
			},
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "StatefulSetList",
			},
			listOptions: []client.ListOption{
				client.InNamespace(ctx.namespaceName),
			},
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "DaemonSetList",
			},
			listOptions: []client.ListOption{
				client.InNamespace(ctx.namespaceName),
			},
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "dynatrace.com",
				Version: "v1beta1",
				Kind:    "DynaKubeList",
			},
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "networking.istio.io",
				Version: "v1alpha3",
				Kind:    "VirtualServiceList",
			},
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "networking.istio.io",
				Version: "v1alpha3",
				Kind:    "ServiceEntryList",
			},
		},
	}
}
