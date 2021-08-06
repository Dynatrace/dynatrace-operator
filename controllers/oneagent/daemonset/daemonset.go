package daemonset

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/statefulset"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/controllers/oneagent"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	labelFeature = "operator.dynatrace.com/feature"

	annotationUnprivileged      = "container.apparmor.security.beta.kubernetes.io/dynatrace-oneagent"
	annotationUnprivilegedValue = "unconfined"
)

type InfraMonitoring struct {
	builderInfo
}

type ClassicFullStack struct {
	builderInfo
}

type builderInfo struct {
	instance      *v1alpha1.DynaKube
	fullstackSpec *v1alpha1.FullStackSpec
	logger        logr.Logger
	clusterId     string
}

type Builder interface {
	BuildDaemonSet() (*appsv1.DaemonSet, error)
}

func NewInfraMonitoring(instance *v1alpha1.DynaKube, logger logr.Logger, clusterId string) Builder {
	return &InfraMonitoring{
		builderInfo{
			instance:      instance,
			fullstackSpec: &instance.Spec.InfraMonitoring.FullStackSpec,
			logger:        logger,
			clusterId:     clusterId,
		},
	}
}

func NewClassicFullStack(instance *v1alpha1.DynaKube, logger logr.Logger, clusterId string) Builder {
	return &InfraMonitoring{
		builderInfo{
			instance:      instance,
			fullstackSpec: &instance.Spec.ClassicFullStack,
			logger:        logger,
			clusterId:     clusterId,
		},
	}
}

func (dsInfo *InfraMonitoring) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := dsInfo.builderInfo.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = dsInfo.instance.Name + fmt.Sprintf("-%s", oneagent.InframonFeature)
	result.Labels[labelFeature] = oneagent.InframonFeature
	result.Spec.Selector.MatchLabels[labelFeature] = oneagent.InframonFeature
	result.Spec.Template.Labels[labelFeature] = oneagent.InframonFeature
}

func (dsInfo *ClassicFullStack) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := dsInfo.builderInfo.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = dsInfo.instance.Name + fmt.Sprintf("-%s", oneagent.ClassicFeature)
	result.Labels[labelFeature] = oneagent.ClassicFeature
	result.Spec.Selector.MatchLabels[labelFeature] = oneagent.ClassicFeature
	result.Spec.Template.Labels[labelFeature] = oneagent.ClassicFeature
}

func (dsInfo *builderInfo) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	instance := dsInfo.instance
	labels := kubeobjects.MergeLabels(dsInfo.buildLabels(), dsInfo.fullstackSpec.Labels)
	maxUnavailable := intstr.FromInt(instance.FeatureOneAgentMaxUnavailable())
	annotations := map[string]string{
		statefulset.AnnotationVersion: instance.Status.OneAgent.Version,
	}

	if dsInfo.fullstackSpec.UseUnprivilegedMode == nil || *dsInfo.fullstackSpec.UseUnprivilegedMode {
		annotations[annotationUnprivileged] = annotationUnprivilegedValue
	}

	result := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: dsInfo.buildLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				// Spec: spec
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
			},
		},
	}

	return result, nil
}

func (dsInfo *builderInfo) buildLabels() map[string]string {
	return map[string]string{
		"dynatrace.com/component":         "operator",
		"operator.dynatrace.com/instance": dsInfo.instance.Name,
	}
}
