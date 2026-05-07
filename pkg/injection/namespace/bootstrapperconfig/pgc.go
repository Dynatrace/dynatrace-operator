package bootstrapperconfig

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	_ = 1 << (10 * iota) //nolint:mnd
	KiB
)

const (
	declarativeInputFileName = "declarative.cbor"
	declarativeWarnSizeBytes = 800 * KiB
	declarativeMaxSizeBytes  = 900 * KiB

	annotationPGCETag = api.InternalFlagPrefix + "pgc-etag"
)

func (s *SecretGenerator) addPGC(ctx context.Context, dk *dynakube.DynaKube, data map[string][]byte, annotations map[string]string) error {
	pgc, err := s.preparePGC(ctx, dk)
	if err != nil {
		return err
	}

	if pgc != nil && len(pgc.Data) != 0 {
		data[declarativeInputFileName] = pgc.Data
		annotations[annotationPGCETag] = pgc.ETag
	}

	return nil
}

func (s *SecretGenerator) preparePGC(ctx context.Context, dk *dynakube.DynaKube) (*oneagent.ProcessGroupConfig, error) {
	log := logd.FromContext(ctx)

	if dk.Status.KubernetesClusterMEID == "" {
		log.Info("kubernetesClusterMEID not available, skipping processgroupingconfig")

		return nil, nil //nolint:nilnil
	}

	cachedPGC := s.readCachedPGC(ctx, dk)

	pgc, err := s.dtClient.GetProcessGroupingConfig(ctx, dk.Status.KubernetesClusterMEID, cachedPGC.ETag)
	if core.HasStatusCode(err, http.StatusNotModified) {
		return cachedPGC, nil
	}

	if err != nil {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	if pgc == nil {
		return nil, nil //nolint:nilnil
	}

	size := len(pgc.Data)
	if size > declarativeMaxSizeBytes {
		log.Error(nil, "DPG API response exceeds maximum size, skipping processgroupingconfig", "size", size, "maxSize", declarativeMaxSizeBytes)

		return nil, nil //nolint:nilnil
	}

	if size > declarativeWarnSizeBytes {
		log.Info("DPG API response exceeds warning size threshold", "size", size, "warnSize", declarativeWarnSizeBytes)
	}

	return pgc, nil
}

func (s *SecretGenerator) readCachedPGC(ctx context.Context, dk *dynakube.DynaKube) (*oneagent.ProcessGroupConfig) {
	var secret corev1.Secret

	key := types.NamespacedName{Name: GetSourceConfigSecretName(dk.Name), Namespace: dk.Namespace}
	_ = s.apiReader.Get(ctx, key, &secret)

	return &oneagent.ProcessGroupConfig{
		ETag: secret.Annotations[annotationPGCETag],
		Data: secret.Data[declarativeInputFileName],
	}
}
