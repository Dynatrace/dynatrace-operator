package mutation

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
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
	enabled bool
}

func NewFeature(ftype FeatureType, enabled bool) Feature {
	return Feature{ftype: ftype, enabled: enabled}
}

func (f Feature) annotationValue() string {
	return strconv.FormatBool(f.enabled)
}

func (f FeatureType) name() string {
	anno := "unknown"
	switch f {
	case OneAgent:
		anno = dtwebhook.OneAgentPrefix
	case DataIngest:
		anno = dtwebhook.DataIngestPrefix
	}
	return anno
}

// for testing only
func (f FeatureType) namePrefixed() string {
	anno := "unknown"
	switch f {
	case OneAgent:
		anno = dtwebhook.AnnotationOneAgentInject
	case DataIngest:
		anno = dtwebhook.AnnotationDataIngestInject
	}
	return anno
}

type InjectionInfo struct {
	features map[FeatureType]bool
}

func NewInjectionInfoForPod(pod *corev1.Pod) *InjectionInfo {
	oneAgentInject := kubeobjects.GetFieldBool(pod.Annotations, dtwebhook.AnnotationOneAgentInject, true)
	dataIngestInject := kubeobjects.GetFieldBool(pod.Annotations, dtwebhook.AnnotationDataIngestInject, oneAgentInject)

	injectionInfo := NewInjectionInfo()
	injectionInfo.add(NewFeature(OneAgent, oneAgentInject))
	injectionInfo.add(NewFeature(DataIngest, dataIngestInject))
	return injectionInfo
}

func NewInjectionInfo() *InjectionInfo {
	return &InjectionInfo{features: make(map[FeatureType]bool)}
}

func (info *InjectionInfo) exists(wanted FeatureType) bool {
	_, exists := info.features[wanted]
	return exists
}

func (info *InjectionInfo) enabled(wanted FeatureType) bool {
	val, exists := info.features[wanted]
	return exists && val
}

func (info *InjectionInfo) anyEnabled() bool {
	for _, enabled := range info.features {
		if enabled {
			return true
		}
	}
	return false
}

func (info *InjectionInfo) add(f Feature) {
	info.features[f.ftype] = f.enabled
}

// for testing only
func (info *InjectionInfo) createInjectAnnotations() map[string]string {
	m := make(map[string]string)
	for featureType, enabled := range info.features {
		f := NewFeature(featureType, enabled)
		m[f.ftype.namePrefixed()] = f.annotationValue()
	}

	return m
}

func (info *InjectionInfo) injectedAnnotation() string {
	builder := strings.Builder{}

	var ftrs []string
	for injectable, enabled := range info.features {
		if enabled {
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

func (info *InjectionInfo) fillAnnotations(pod *corev1.Pod) {
	injectedAnnotation := info.injectedAnnotation()
	if injectedAnnotation != "" {
		if pod.Annotations == nil {
			pod.Annotations = map[string]string{}
		}
		pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = injectedAnnotation
	}
}
