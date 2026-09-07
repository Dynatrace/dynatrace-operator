// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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
	ExperimentalEnableKubemonOperand     = "EXPERIMENTAL_ENABLE_KUBEMON_OPERAND"
	ExperimentalEnablePrometheus         = "EXPERIMENTAL_ENABLE_PROMETHEUS"

	DTClientCacheCleanInterval        = "DT_CLIENT_CACHE_CLEAN_INTERVAL"
	defaultDTClientCacheCleanInterval = time.Hour
	minDTClientCacheCleanInterval     = 5 * time.Minute
	maxDTClientCacheCleanInterval     = 100 * time.Hour

	DTClientConnectionTimeoutEnvVar = "DT_CLIENT_CONNECTION_TIMEOUT"
	// DefaultCSIDriverDTClientConnectionTimeout enough time to download an OneAgent package of about 1GB in size
	DefaultCSIDriverDTClientConnectionTimeout = maxDTClientConnectionTimeout
	DefaultOperatorDTClientConnectionTimeout  = minDTClientConnectionTimeout
	minDTClientConnectionTimeout              = 30 * time.Second
	maxDTClientConnectionTimeout              = 15 * time.Minute

	DefaultRequeueAfterEnvVar = "DT_DEFAULT_REQUEUE_AFTER"
	defaultRequeueInterval    = 15 * time.Minute
	minRequeueInterval        = time.Minute
	maxRequeueInterval        = time.Hour

	WebhookCertsRequeueAfterEnvVar  = "DT_WEBHOOK_CERTS_REQUEUE_AFTER"
	defaultWebhookCertsRequeueAfter = 3 * time.Hour
	minWebhookCertsRequeueAfter     = 5 * time.Minute
	maxWebhookCertsRequeueAfter     = 11 * time.Hour

	WebhookCertsRenewalThresholdEnvVar  = "DT_WEBHOOK_CERTS_RENEWAL_THRESHOLD"
	defaultWebhookCertsRenewalThreshold = 12 * time.Hour
	minWebhookCertsRenewalThreshold     = 12 * time.Hour // must be >= minCertificateRenewalThreshold (pkg/controllers/certificates)
	maxWebhookCertsRenewalThreshold     = 720 * time.Hour

	WebhookCertsServerDurationEnvVar  = "DT_WEBHOOK_CERTS_SERVER_DURATION"
	defaultWebhookCertsServerDuration = 7 * 24 * time.Hour
	minWebhookCertsServerDuration     = 24 * time.Hour
	maxWebhookCertsServerDuration     = 365 * 24 * time.Hour

	WebhookCertsRootDurationEnvVar  = "DT_WEBHOOK_CERTS_ROOT_DURATION"
	defaultWebhookCertsRootDuration = 365 * 24 * time.Hour
	minWebhookCertsRootDuration     = 7 * 24 * time.Hour
	maxWebhookCertsRootDuration     = 10 * 365 * 24 * time.Hour

	WebhookMetadataSizeLimitEnvVar       = "DT_METADATA_SIZE_LIMIT"
	defaultWebhookMetadataSizeLimitValue = 24 * 1024
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
	return getSafeDurationFromEnv(ctx, DefaultRequeueAfterEnvVar, defaultRequeueInterval, minRequeueInterval, maxRequeueInterval)
}

func GetDTClientCacheCleanInterval(ctx context.Context) time.Duration {
	return getSafeDurationFromEnv(ctx, DTClientCacheCleanInterval, defaultDTClientCacheCleanInterval, minDTClientCacheCleanInterval, maxDTClientCacheCleanInterval)
}

func GetOperatorDTClientConnectionTimeout(ctx context.Context) time.Duration {
	return getSafeDurationFromEnv(ctx, DTClientConnectionTimeoutEnvVar, DefaultOperatorDTClientConnectionTimeout, minDTClientConnectionTimeout, maxDTClientConnectionTimeout)
}

func GetCSIDriverDTClientConnectionTimeout(ctx context.Context) time.Duration {
	return getSafeDurationFromEnv(ctx, DTClientConnectionTimeoutEnvVar, DefaultCSIDriverDTClientConnectionTimeout, minDTClientConnectionTimeout, maxDTClientConnectionTimeout)
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

func IsKubemonOperandEnabled() bool {
	rawValue := os.Getenv(ExperimentalEnableKubemonOperand)
	if rawValue == "" {
		return false
	}

	value, err := strconv.ParseBool(rawValue)
	if err != nil {
		return false
	}

	return value
}

func IsPrometheusEnabled() bool {
	rawValue := os.Getenv(ExperimentalEnablePrometheus)
	if rawValue == "" {
		return false
	}

	value, err := strconv.ParseBool(rawValue)
	if err != nil {
		return false
	}

	return value
}

func NewRef(envName string) string {
	return fmt.Sprintf("$(%s)", envName)
}

func GetWebhookCertsRequeueAfter(ctx context.Context) time.Duration {
	return getSafeDurationFromEnv(ctx, WebhookCertsRequeueAfterEnvVar, defaultWebhookCertsRequeueAfter, minWebhookCertsRequeueAfter, maxWebhookCertsRequeueAfter)
}

func GetWebhookCertsRenewalThreshold(ctx context.Context) time.Duration {
	return getSafeDurationFromEnv(ctx, WebhookCertsRenewalThresholdEnvVar, defaultWebhookCertsRenewalThreshold, minWebhookCertsRenewalThreshold, maxWebhookCertsRenewalThreshold)
}

func GetWebhookCertsServerDuration(ctx context.Context) time.Duration {
	return getSafeDurationFromEnv(ctx, WebhookCertsServerDurationEnvVar, defaultWebhookCertsServerDuration, minWebhookCertsServerDuration, maxWebhookCertsServerDuration)
}

func GetWebhookCertsRootDuration(ctx context.Context) time.Duration {
	return getSafeDurationFromEnv(ctx, WebhookCertsRootDurationEnvVar, defaultWebhookCertsRootDuration, minWebhookCertsRootDuration, maxWebhookCertsRootDuration)
}

func getSafeDurationFromEnv(ctx context.Context, envVar string, defaultValue, minValue, maxValue time.Duration) time.Duration {
	_, log := logd.NewFromContext(ctx, "k8senv")

	rawDuration := os.Getenv(envVar)
	if rawDuration == "" {
		log.Debug("no custom env set, using default", "env", envVar, "default", defaultValue)

		return defaultValue
	}

	duration, err := time.ParseDuration(rawDuration)
	if err != nil {
		log.Error(err, "invalid duration value, using default", "env", envVar, "value", rawDuration, "default", defaultValue)

		return defaultValue
	}

	if duration < minValue || duration > maxValue {
		log.Info("duration not in allowed range, using default", "env", envVar, "value", duration, "min", minValue, "max", maxValue, "default", defaultValue)

		return defaultValue
	}

	log.Info("using custom duration", "env", envVar, "value", duration)

	return duration
}

func GetMetadaSizeLimit(ctx context.Context) int {
	var log logd.Logger
	if ctx != nil {
		_, log = logd.NewFromContext(ctx, "k8senv")
	}

	rawValue := os.Getenv(WebhookMetadataSizeLimitEnvVar)
	if rawValue == "" {
		if ctx != nil {
			log.Debug("no custom env set, using default", "env", WebhookMetadataSizeLimitEnvVar, "default", defaultWebhookMetadataSizeLimitValue)
		}

		return defaultWebhookMetadataSizeLimitValue
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil || value < 0 {
		if ctx != nil {
			log.Error(err, "invalid int value, using default", "env", WebhookMetadataSizeLimitEnvVar, "value", rawValue, "default", defaultWebhookMetadataSizeLimitValue)
		}

		return defaultWebhookMetadataSizeLimitValue
	}

	if ctx != nil {
		log.Info("using custom int value", "env", WebhookMetadataSizeLimitEnvVar, "value", value)
	}

	return value
}

// AppendGoMemoryLimit appends GOMEMLIMIT env var
func AppendGoMemoryLimit(envs []corev1.EnvVar, resources corev1.ResourceRequirements) []corev1.EnvVar {
	if memLimit := resources.Limits.Memory(); !memLimit.IsZero() {
		gomemlimit := memLimit.Value() / 10 * 9 //nolint:mnd // 90%
		envs = append(envs, corev1.EnvVar{Name: "GOMEMLIMIT", Value: strconv.FormatInt(gomemlimit, 10)})
	}

	return envs
}
