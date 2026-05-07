package bootstrapperconfig

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
)

const (
	_ = 1 << (10 * iota) //nolint:mnd
	KiB
)

const (
	declarativeInputFileName = "declarative.cbor"
	declarativeWarnSizeBytes = 700 * KiB
	declarativeMaxSizeBytes  = 880 * KiB
)

func (s *SecretGenerator) addPGC(ctx context.Context, dk *dynakube.DynaKube, data map[string][]byte) error {
	config, err := s.preparePGC(ctx, dk)
	if err != nil {
		return err
	}

	if len(config) != 0 {
		data[declarativeInputFileName] = config
	}

	return nil
}

func (s *SecretGenerator) preparePGC(ctx context.Context, dk *dynakube.DynaKube) ([]byte, error) {
	log := logd.FromContext(ctx)

	pgc, err := s.dtClient.GetProcessGroupingConfig(ctx, dk.Status.KubernetesClusterMEID, "")
	if err != nil {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	if pgc == nil {
		return nil, nil
	}

	size := len(pgc.Data)
	if size > declarativeMaxSizeBytes {
		log.Error(nil, "DPG API response exceeds maximum size, skipping declarative.cbor", "size", size, "maxSize", declarativeMaxSizeBytes)

		return nil, nil
	}

	if size > declarativeWarnSizeBytes {
		log.Info("DPG API response exceeds warning size threshold", "size", size, "warnSize", declarativeWarnSizeBytes)
	}

	return pgc.Data, nil
}
