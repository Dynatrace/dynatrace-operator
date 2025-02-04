package scenarios

import (
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func FilterFeatures(cfg envconf.Config, feats []features.Feature) (filtered []features.Feature) {
	if cfg.FeatureRegex() != nil {
		for _, feat := range feats {
			if cfg.FeatureRegex().Match([]byte(feat.Name())) {
				filtered = append(filtered, feat)
			}
		}
	} else {
		filtered = feats
	}

	return
}
