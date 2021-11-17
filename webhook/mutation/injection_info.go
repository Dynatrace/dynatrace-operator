package mutation

import (
	"sort"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/webhook"
)

type Injectable interface {
	name() string
	annotationValue() string
}

type FeatureType int

const (
	OneAgent FeatureType = iota
	DataIngest
)

type Feature struct {
	ftype   FeatureType
	Enabled bool
}

func NewFeature(ftype FeatureType, enabled bool) *Feature {
	return &Feature{ftype: ftype, Enabled: enabled}
}

func (f Feature) annotationValue() string {
	if f.Enabled {
		return "true"
	}
	return "false"
}

func (f Feature) name() string {
	anno := "unknown"
	switch f.ftype {
	case OneAgent:
		anno = webhook.OneAgentPrefix
	case DataIngest:
		anno = webhook.DataIngestPrefix
	}
	return anno
}

type InjectionInfo struct {
	features map[*Feature]struct{}
}

func NewInjectionInfo() *InjectionInfo {
	return &InjectionInfo{features: make(map[*Feature]struct{})}
}

func (info *InjectionInfo) enabled(wanted FeatureType) bool {
	for k := range info.features {
		if k.ftype == wanted && k.Enabled {
			return true
		}
	}
	return false
}

func (info *InjectionInfo) any() bool {
	for k := range info.features {
		if k.Enabled {
			return true
		}
	}
	return false
}

func (info *InjectionInfo) add(f *Feature) {
	info.features[f] = struct{}{}
}

func (info *InjectionInfo) injectedAnnotation() string {
	builder := strings.Builder{}

	ftrs := []string{}
	for injectable := range info.features {
		if injectable.Enabled {
			ftrs = append(ftrs, injectable.name())
		}
	}

	sort.Strings(ftrs)

	for _, ftr := range ftrs {
		builder.WriteString(ftr)
		builder.WriteRune(',')
	}

	ret := builder.String()
	if len(ret) > 0 {
		return ret[:len(ret)-1]
	} else {
		return ret
	}
}
