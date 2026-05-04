package bootstrapperconfig

import (
	"bytes"
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
)

const (
	declarativeInputFileName = "declarative.cbor"
	declarativeWarnSizeBytes = 800 * 1024 // 800 KiB
	declarativeMaxSizeBytes  = 980 * 1024 // 980 KiB
)

func (s *SecretGenerator) addDeclarativeConfig(ctx context.Context, dk *dynakube.DynaKube, data map[string][]byte) error {
	config, err := s.prepareDeclarativeConfig(ctx, dk)
	if err != nil {
		return err
	}

	if len(config) != 0 {
		data[declarativeInputFileName] = config
	}

	return nil
}

func (s *SecretGenerator) prepareDeclarativeConfig(ctx context.Context, dk *dynakube.DynaKube) ([]byte, error) {
	log := logd.FromContext(ctx)

	var buf bytes.Buffer

	_, err := s.dtClient.GetProcessGroupingConfig(ctx, dk.Status.KubeSystemUUID, "", &buf)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	size := buf.Len()
	if size > declarativeMaxSizeBytes {
		log.Error(nil, "DPG API response exceeds maximum size, skipping declarative.cbor", "size", size, "maxSize", declarativeMaxSizeBytes)

		return nil, nil
	}

	if size > declarativeWarnSizeBytes {
		log.Info("DPG API response exceeds warning size threshold", "size", size, "warnSize", declarativeWarnSizeBytes)
	}

	return buf.Bytes(), nil
}
