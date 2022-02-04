package v1beta1

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const InternalFlagPrefix = "internal.operator.dynatrace.com/"

func GetInternalFlags(obj metav1.Object) map[string]string {
	internalAnnotations := make(map[string]string)
	for annotation, value := range obj.GetAnnotations() {
		if strings.HasPrefix(annotation, InternalFlagPrefix) {
			internalAnnotations[annotation] = value
		}
	}
	return internalAnnotations
}

func IsInternalFlagsEqual(obj1, obj2 metav1.Object) bool {
	return fmt.Sprint(GetInternalFlags(obj1)) == fmt.Sprint(GetInternalFlags(obj2))
}
