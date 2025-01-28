package daemonset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"k8s.io/apimachinery/pkg/types"
)

const annotationTenantTokenHash = api.InternalFlagPrefix + "tenant-token-hash"

func (r *Reconciler) calculateTenantTokenHash(ctx context.Context) (string, error) {
	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{
		Name:      r.dk.OneagentTenantSecret(),
		Namespace: r.dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)

	if err != nil {
		log.Error(err, "secret for tenant token was not available at DaemonSet build time", "dynakube", r.dk.Name)
		conditions.SetKubeApiError(r.dk.Conditions(), conditionType, err)
	}

	return hasher.GenerateHash(tenantToken)
}

func (r *Reconciler) buildAnnotations(ctx context.Context) (map[string]string, error) {
	hash, err := r.calculateTenantTokenHash(ctx)

	annotations := map[string]string{
		annotationTenantTokenHash: hash,
	}

	return annotations, err
}
