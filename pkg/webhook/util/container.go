package util

import (
	"strings"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
)

func checkInjectionAnnotation(annotations map[string]string, name string) bool {
	for key, value := range annotations {
		if strings.HasPrefix(key, dtwebhook.AnnotationContainerInjection) {
			keySplit := strings.Split(key, "/")
			if len(keySplit) == 2 && keySplit[1] == name {
				return value == "false"
			}
		}
	}

	return false
}

func IsContainerExcludedFromInjection(request *dtwebhook.BaseRequest, name string) bool {
	return checkInjectionAnnotation(request.DynaKube.Annotations, name) || checkInjectionAnnotation(request.Pod.Annotations, name)
}
