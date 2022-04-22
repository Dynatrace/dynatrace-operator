package capability

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/statefulset"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const extensionsRuntimeProperties = dynatracev1beta1.AnnotationFeaturePrefix + "extensions."

type EecRuntimeConfig struct {
	Revision   int               `json:"revision"`
	BooleanMap map[string]bool   `json:"booleanMap"`
	StringMap  map[string]string `json:"stringMap"`
	LongMap    map[string]int64  `json:"longMap"`
}

func NewEecRuntimeConfig() *EecRuntimeConfig {
	return &EecRuntimeConfig{
		Revision:   1,
		BooleanMap: make(map[string]bool),
		StringMap:  make(map[string]string),
		LongMap:    make(map[string]int64),
	}
}

func getExtensionsFlagsFromAnnotations(instance *dynatracev1beta1.DynaKube) map[string]string {
	extensionsFlags := make(map[string]string)
	for flag, val := range dynatracev1beta1.FlagsWithPrefix(instance, extensionsRuntimeProperties) {
		runtimeProp := strings.TrimPrefix(flag, extensionsRuntimeProperties)
		extensionsFlags[runtimeProp] = val
	}
	return extensionsFlags
}

func buildEecRuntimeConfig(instance *dynatracev1beta1.DynaKube) *EecRuntimeConfig {
	eecRuntimeConfig := NewEecRuntimeConfig()

	for runtimeProp, val := range getExtensionsFlagsFromAnnotations(instance) {
		if parsedLongInt, err := strconv.ParseInt(val, 10, 64); err == nil {
			eecRuntimeConfig.LongMap[runtimeProp] = parsedLongInt
		} else if parsedBool, err := strconv.ParseBool(val); err == nil {
			eecRuntimeConfig.BooleanMap[runtimeProp] = parsedBool
		} else {
			eecRuntimeConfig.StringMap[runtimeProp] = val
		}
	}

	return eecRuntimeConfig
}

func buildEecRuntimeConfigJson(instance *dynatracev1beta1.DynaKube) (string, error) {
	runtimeConfiguration, err := json.Marshal(buildEecRuntimeConfig(instance))
	if err != nil {
		log.Error(err, "problem serializing map with EEC runtime properties")
		return "", err
	}
	return string(runtimeConfiguration), nil
}

func CreateEecConfigMap(instance *dynatracev1beta1.DynaKube, feature string) (*corev1.ConfigMap, error) {
	eecRuntimeConfigurationJson, err := buildEecRuntimeConfigJson(instance)
	if err != nil {
		return nil, err
	}

	if len(instance.Name) == 0 || len(feature) == 0 {
		return nil, fmt.Errorf("empty instance or module name not allowed (instance: %s, module: %s)", instance.Name, feature)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulset.BuildEecConfigMapName(instance.Name, feature),
			Namespace: instance.Namespace,
		},
		Data: map[string]string{
			"runtimeConfiguration": eecRuntimeConfigurationJson,
		},
	}, nil
}
