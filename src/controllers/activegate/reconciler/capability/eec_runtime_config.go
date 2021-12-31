package capability

import (
	"encoding/json"
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/statefulset"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const extensionsRuntimeProperties = dynatracev1beta1.InternalFlagPrefix + "extensions."

func getExtensionsFlagsFromAnnotations(instance *dynatracev1beta1.DynaKube) map[string]string {
	extensionsFlags := make(map[string]string)
	for flag, val := range dynatracev1beta1.GetInternalFlags(instance) {
		if strings.HasPrefix(flag, extensionsRuntimeProperties) {
			runtimeProp := strings.TrimPrefix(flag, extensionsRuntimeProperties)
			extensionsFlags[runtimeProp] = val
		}
	}
	return extensionsFlags
}

func buildEecRuntimeConfig(instance *dynatracev1beta1.DynaKube) map[string]interface{} {
	booleanMap := make(map[string]bool)
	stringMap := make(map[string]string)
	longMap := make(map[string]int64)

	for runtimeProp, val := range getExtensionsFlagsFromAnnotations(instance) {
		if parsedLongInt, err := strconv.ParseInt(val, 10, 64); err == nil {
			longMap[runtimeProp] = parsedLongInt
		} else if parsedBool, err := strconv.ParseBool(val); err == nil {
			booleanMap[runtimeProp] = parsedBool
		} else {
			stringMap[runtimeProp] = val
		}
	}

	return map[string]interface{}{
		"revision":   1,
		"booleanMap": booleanMap,
		"stringMap":  stringMap,
		"longMap":    longMap,
	}
}

func buildEecRuntimeConfigJson(instance *dynatracev1beta1.DynaKube) (string, error) {
	runtimeConfiguration, err := json.Marshal(buildEecRuntimeConfig(instance))
	if err != nil {
		log.Error(err, "problem serializing map with runtime properties")
		return "", err
	}
	return string(runtimeConfiguration), nil
}

func CreateEecConfigMap(instance *dynatracev1beta1.DynaKube, feature string) *corev1.ConfigMap {
	if !instance.NeedsStatsd() {
		return nil
	}

	eecRuntimeConfigurationJson, err := buildEecRuntimeConfigJson(instance)
	if err != nil {
		log.Error(err, "failed to build EEC runtime configuration JSON")
		return nil
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulset.BuildEecConfigMapName(instance.Name, feature),
			Namespace: instance.Namespace,
		},
		Data: map[string]string{
			"runtimeConfiguration": eecRuntimeConfigurationJson,
		},
	}
}
