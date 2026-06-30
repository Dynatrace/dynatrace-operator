package k8senv

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	corev1 "k8s.io/api/core/v1"
)

const (
	NodeName                    = "KUBE_NODE_NAME"
	CSIDataDir                  = "CSI_DATA_DIR"
	PodNamespace                = "POD_NAMESPACE"
	PodName                     = "POD_NAME"
	DTOperatorImageEnvName      = "DT_OPERATOR_IMAGE"
	DTOperatorPullSecretEnvName = "DT_OPERATOR_PULL_SECRET"
	OLMOperatorNamespaceEnv     = "OLM_OPERATOR_NAMESPACE"
	AppVersion                  = "APP_VERSION"

	DTExtractCodeModulesImageLinksEnvVar = "DT_EXTRACT_CODEMODULES_IMAGE_LINKS"

	DTClientCacheCleanInterval        = "DT_CLIENT_CACHE_CLEAN_INTERVAL"
	defaultDTClientCacheCleanInterval = time.Hour
	minDTClientCacheCleanInterval     = 5 * time.Minute
	maxDTClientCacheCleanInterval     = 100 * time.Hour

	DefaultRequeueAfterEnvVar = "DT_DEFAULT_REQUEUE_AFTER"
	defaultRequeueInterval    = 15 * time.Minute
	minRequeueInterval        = time.Minute
	maxRequeueInterval        = time.Hour
)

func Find(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	for i, envVar := range envVars {
		if envVar.Name == name {
			// returning reference to env var to ease later manipulation of it
			return &envVars[i]
		}
	}

	return nil
}

func FindCaseInsensitive(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	for i, envVar := range envVars {
		if strings.EqualFold(envVar.Name, name) {
			// returning reference to env var to ease later manipulation of it
			return &envVars[i]
		}
	}

	return nil
}

func Contains(envVars []corev1.EnvVar, envVarToCheck string) bool {
	for _, envVar := range envVars {
		if envVar.Name == envVarToCheck {
			return true
		}
	}

	return false
}

func Append(envVars []corev1.EnvVar, envVarToAppend corev1.EnvVar) ([]corev1.EnvVar, bool) {
	added := false

	if !Contains(envVars, envVarToAppend.Name) {
		envVars = append(envVars, envVarToAppend)
		added = true
	}

	return envVars, added
}

func AddOrUpdate(envVars []corev1.EnvVar, desiredEnvVar corev1.EnvVar) []corev1.EnvVar {
	targetEnvVar := Find(envVars, desiredEnvVar.Name)
	if targetEnvVar != nil {
		*targetEnvVar = desiredEnvVar
	} else {
		envVars = append(envVars, desiredEnvVar)
	}

	return envVars
}

func NewSourceForField(fieldPath string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: fieldPath}}
}

func DefaultNamespace() string {
	namespace := os.Getenv(PodNamespace)

	if namespace == "" {
		return "dynatrace"
	}

	return namespace
}

func GetNodeName() string {
	return os.Getenv(NodeName)
}

func GetCSIDataDir() string {
	return os.Getenv(CSIDataDir)
}

func GetDefaultRequeueAfter(ctx context.Context) time.Duration {
	_, log := logd.NewFromContext(ctx, "k8senv")

	rawDuration := os.Getenv(DefaultRequeueAfterEnvVar)
	if rawDuration == "" {
		log.Debug("no custom env set, using default", "env", DefaultRequeueAfterEnvVar, "default", defaultRequeueInterval)

		return defaultRequeueInterval
	}

	duration, err := time.ParseDuration(rawDuration)
	if err != nil {
		log.Error(err, "failed to parse default requeue interval, fallback to", "duration", rawDuration)

		return defaultRequeueInterval
	}

	if duration < minRequeueInterval || duration > maxRequeueInterval {
		log.Info("requeueAfter from env is not in the allowed range, using default", "env", DefaultRequeueAfterEnvVar, "value", duration, "min", minRequeueInterval, "max", maxRequeueInterval, "default", defaultRequeueInterval)

		return defaultRequeueInterval
	}

	return duration
}

func GetDTClientCacheCleanInterval(ctx context.Context) time.Duration {
	_, log := logd.NewFromContext(ctx, "k8senv")

	rawDuration := os.Getenv(DTClientCacheCleanInterval)
	if rawDuration == "" {
		log.Debug("no custom env set, using default", "env", DTClientCacheCleanInterval, "default", defaultDTClientCacheCleanInterval)

		return defaultDTClientCacheCleanInterval
	}

	parsedDuration, err := time.ParseDuration(rawDuration)
	if err != nil {
		log.Info("couldn't parse time.Duration from env", "env", DTClientCacheCleanInterval, "value", rawDuration, "err", err)

		return defaultDTClientCacheCleanInterval
	}

	if parsedDuration < minDTClientCacheCleanInterval || parsedDuration > maxDTClientCacheCleanInterval {
		log.Info("parsed time.Duration from env is not in the allowed range", "env", DTClientCacheCleanInterval, "value", parsedDuration, "min", minDTClientCacheCleanInterval, "max", maxDTClientCacheCleanInterval)

		return defaultDTClientCacheCleanInterval
	}

	return parsedDuration
}

// GetDTExtractCodeModulesImageLinks reads the value of DT_EXTRACT_CODEMODULES_IMAGE_LINKS.
func GetDTExtractCodeModulesImageLinks(ctx context.Context) bool {
	rawValue := os.Getenv(DTExtractCodeModulesImageLinksEnvVar)
	if rawValue == "" {
		return false
	}

	value, err := strconv.ParseBool(rawValue)
	if err != nil {
		_, log := logd.NewFromContext(ctx, "k8senv")
		log.Info("couldn't parse bool from env", "env", DTExtractCodeModulesImageLinksEnvVar, "value", rawValue, "err", err)

		return false
	}

	return value
}

func NewRef(envName string) string {
	return fmt.Sprintf("$(%s)", envName)
}
