package namespacesmapper

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
	codeModulesMapName = "code-modules-map"
	dataIngestMapName  = "data-ingest-map"
)

type dynaKubeFunc func(dk dynatracev1alpha1.DynaKube) bool
type selectorFunc func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector

// getOrCreateMap returns ConfigMap in operator's namespace
func getOrCreateMap(ctx context.Context, clt client.Client, operatorNs string, cfgmapName string) (*corev1.ConfigMap, error) {
	var cfgmap corev1.ConfigMap
	if err := clt.Get(ctx, client.ObjectKey{Name: cfgmapName, Namespace: operatorNs}, &cfgmap); err != nil {
		if k8serrors.IsNotFound(err) {
			nsmap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cfgmapName, Namespace: operatorNs},
			}

			if err := clt.Create(ctx, &nsmap); err != nil {
				if !k8serrors.IsAlreadyExists(err) {
					return nil, errors.WithMessagef(err, "failed to create ConfigMap %s", cfgmapName)
				}
			}

			if err := clt.Get(ctx, client.ObjectKey{Name: cfgmapName, Namespace: operatorNs}, &cfgmap); err != nil {
				return nil, errors.WithMessagef(err, "ConfigMap %s created. Failed to query the map", cfgmapName)
			}
		} else {
			return nil, errors.WithMessagef(err, "failed to query ConfigMap %s", cfgmapName)
		}
	}
	return &cfgmap, nil
}
