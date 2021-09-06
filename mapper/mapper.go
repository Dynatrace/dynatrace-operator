package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CodeModulesMapName = "code-modules-map"
	DataIngestMapName  = "data-ingest-map"
	ReadyLabelKey      = "dynatrace.com/ns"
)

type dynaKubeFilterFunc func(dk dynatracev1alpha1.DynaKube) bool
type namespaceSelectorFunc func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector

// getOrCreateMap returns ConfigMap in operator's namespace
func getOrCreateMap(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, cfgMapName string) (*corev1.ConfigMap, error) {
	var cfgMap corev1.ConfigMap
	if err := apiReader.Get(ctx, client.ObjectKey{Name: cfgMapName, Namespace: operatorNs}, &cfgMap); err != nil {
		if k8serrors.IsNotFound(err) {
			cfgMap = corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cfgMapName, Namespace: operatorNs},
			}

			if err := clt.Create(ctx, &cfgMap); err != nil {
				if !k8serrors.IsAlreadyExists(err) {
					return nil, errors.WithMessagef(err, "failed to create ConfigMap %s", cfgMapName)
				}
			}

			if err := apiReader.Get(ctx, client.ObjectKey{Name: cfgMapName, Namespace: operatorNs}, &cfgMap); err != nil {
				return nil, errors.WithMessagef(err, "ConfigMap %s created. Failed to query the map", cfgMapName)
			}
		} else {
			return nil, errors.WithMessagef(err, "failed to query ConfigMap %s", cfgMapName)
		}
	}
	return &cfgMap, nil
}

func updateNamespaceLabel(ctx context.Context, clt client.Client, operatorNs string, ns *corev1.Namespace, dk *dynatracev1alpha1.DynaKube) {
	if operatorNs == ns.Name {
		return
	}
	if dkName, ok := ns.Labels[ReadyLabelKey]; ok && dkName == dk.Name {
		return
	}
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels[ReadyLabelKey] = dk.Name
}

func removeNamespaceLabel(ctx context.Context, clt client.Client, ns *corev1.Namespace) {
	if _, ok := ns.Labels[ReadyLabelKey]; !ok {
		return
	}
	delete(ns.Labels, ReadyLabelKey)
}
