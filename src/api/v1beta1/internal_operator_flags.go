package v1beta1

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const InternalFlagPrefix = "internal.operator.dynatrace.com/"

func FlagsWithPrefix(obj metav1.Object, prefix string) map[string]string {
	filteredAnnotations := make(map[string]string)
	for annotation, value := range obj.GetAnnotations() {
		if strings.HasPrefix(annotation, prefix) {
			filteredAnnotations[annotation] = value
		}
	}
	return filteredAnnotations
}

func InternalFlags(obj metav1.Object) map[string]string {
	return FlagsWithPrefix(obj, InternalFlagPrefix)
}

func IsInternalFlagsEqual(obj1, obj2 metav1.Object) bool {
	return fmt.Sprint(InternalFlags(obj1)) == fmt.Sprint(InternalFlags(obj2))
}
