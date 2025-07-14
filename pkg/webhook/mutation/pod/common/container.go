package common

import (
	"strings"
)

func checkInjectionAnnotation(annotations map[string]string, name string) bool {
	for key, value := range annotations {
		if strings.HasPrefix(key, AnnotationContainerInjection) {
			keySplit := strings.Split(key, "/")
			if len(keySplit) == 2 && keySplit[1] == name {
				return value == "false"
			}
		}
	}

	return false
}

func IsContainerExcludedFromInjection(dkAnnotations, podAnnotations map[string]string, name string) bool {
	return checkInjectionAnnotation(dkAnnotations, name) || checkInjectionAnnotation(podAnnotations, name)
}
