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

func getOrCreateMap(ctx context.Context, clt client.Client, opns string, cfgmapname string) (*corev1.ConfigMap, error) {
	var cfgmap corev1.ConfigMap
	if err := clt.Get(ctx, client.ObjectKey{Name: cfgmapname, Namespace: opns}, &cfgmap); err != nil {
		if k8serrors.IsNotFound(err) {
			nsmap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cfgmapname, Namespace: opns},
			}

			if err := clt.Create(ctx, &nsmap); err != nil {
				if !k8serrors.IsAlreadyExists(err) {
					return nil, errors.WithMessagef(err, "failed to create ConfigMap %s", cfgmapname)
				}
			}

			if err := clt.Get(ctx, client.ObjectKey{Name: cfgmapname, Namespace: opns}, &cfgmap); err != nil {
				return nil, errors.WithMessagef(err, "ConfigMap %s created. Failed to query the map", cfgmapname)
			}
		} else {
			return nil, errors.WithMessagef(err, "failed to query ConfigMap %s", cfgmapname)
		}
	}
	return &cfgmap, nil
}
