package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	annotationFeatureActiveGate = AnnotationFeaturePrefix + "activegate-"
	annotationFeatureEec        = annotationFeatureActiveGate + "eec-"
	annotationFeatureStatsd     = annotationFeatureActiveGate + "statsd-"
)

// FeatureEecResourcesRequests is a feature flag to define CPU or memory requests for the EEC container
func (dk *DynaKube) FeatureEecResourcesRequests(resourceName corev1.ResourceName) *resource.Quantity {
	return eecResourceRequirements(dk, "requests-"+resourceName)
}

// FeatureEecResourcesLimits is a feature flag to define CPU or memory limits for the EEC container
func (dk *DynaKube) FeatureEecResourcesLimits(resourceName corev1.ResourceName) *resource.Quantity {
	return eecResourceRequirements(dk, "limits-"+resourceName)
}

// FeatureStatsdResourcesRequests is a feature flag to define CPU or memory requests for the StatsD container
func (dk *DynaKube) FeatureStatsdResourcesRequests(resourceName corev1.ResourceName) *resource.Quantity {
	return statsdResourceRequirements(dk, "requests-"+resourceName)
}

// FeatureStatsdResourcesLimits is a feature flag to define CPU or memory limits for the StatsD container
func (dk *DynaKube) FeatureStatsdResourcesLimits(resourceName corev1.ResourceName) *resource.Quantity {
	return statsdResourceRequirements(dk, "limits-"+resourceName)
}

// E.g. "feature.dynatrace.com/activegate-eec-resources-limits-cpu": "100m"
func formatResourceName(resourceName corev1.ResourceName) string {
	return "resources-" + string(resourceName)
}

func eecResourceRequirements(dk *DynaKube, resourceName corev1.ResourceName) *resource.Quantity {
	return resourceRequirements(dk, annotationFeatureEec, resourceName)
}

func statsdResourceRequirements(dk *DynaKube, resourceName corev1.ResourceName) *resource.Quantity {
	return resourceRequirements(dk, annotationFeatureStatsd, resourceName)
}

func resourceRequirements(dk *DynaKube, flagPrefix string, resourceName corev1.ResourceName) *resource.Quantity {
	flagName := flagPrefix + formatResourceName(resourceName)

	val, ok := dk.Annotations[flagName]
	if !ok {
		return nil
	}

	quantity, err := resource.ParseQuantity(val)
	if err != nil {
		log.Info("Problem parsing resource requirements for", "flagName", flagName, "val", val, "err", err.Error())
		return nil
	}

	return &quantity
}

// +kubebuilder:object:generate=false
type ResourceRequirementer interface {
	Requests(corev1.ResourceName) *resource.Quantity
	Limits(corev1.ResourceName) *resource.Quantity
}

func ResourceNames() []corev1.ResourceName {
	return []corev1.ResourceName{
		corev1.ResourceCPU, corev1.ResourceMemory,
	}
}

func BuildResourceRequirements(resourceRequirementer ResourceRequirementer) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{
		Limits:   make(corev1.ResourceList),
		Requests: make(corev1.ResourceList),
	}

	for _, resourceName := range ResourceNames() {
		if quantity := resourceRequirementer.Limits(resourceName); quantity != nil {
			requirements.Limits[resourceName] = *quantity
		}
		if quantity := resourceRequirementer.Requests(resourceName); quantity != nil {
			requirements.Requests[resourceName] = *quantity
		}
	}

	return requirements
}
