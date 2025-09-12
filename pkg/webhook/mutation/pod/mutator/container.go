package mutator

import (
	"strings"
)

func checkInjectionAnnotation(annotations map[string]string, name string) bool {
	for key, value := range annotations {
		if strings.HasPrefix(key, AnnotationContainerInjection) {
			_, path, hasPath := strings.Cut(key, "/")
			if hasPath && path == name {
				return value == "false"
			}
		}
	}

	return false
}

func IsContainerExcludedFromInjection(dkAnnotations, podAnnotations map[string]string, name string) bool {
	return checkInjectionAnnotation(dkAnnotations, name) || checkInjectionAnnotation(podAnnotations, name)
}
