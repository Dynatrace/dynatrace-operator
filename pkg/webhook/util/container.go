package util

import (
	"strings"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
)

func isContainerExcluded(annotations map[string]string, name string) bool {
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

func ContainerIsExcluded(request *dtwebhook.BaseRequest, name string) bool {
	return isContainerExcluded(request.DynaKube.Annotations, name) || isContainerExcluded(request.Pod.Annotations, name)
}
