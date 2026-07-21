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
	KubemonEnableOperand                 = "KUBEMON_ENABLE_OPERAND"

	DTClientCacheCleanInterval        = "DT_CLIENT_CACHE_CLEAN_INTERVAL"
	defaultDTClientCacheCleanInterval = time.Hour
	minDTClientCacheCleanInterval     = 5 * time.Minute
	maxDTClientCacheCleanInterval     = 100 * time.Hour

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
	return parseDuration(ctx, DefaultRequeueAfterEnvVar, defaultRequeueInterval, minRequeueInterval, maxRequeueInterval)
}

func GetDTClientCacheCleanInterval(ctx context.Context) time.Duration {
	return parseDuration(ctx, DTClientCacheCleanInterval, defaultDTClientCacheCleanInterval, minDTClientCacheCleanInterval, maxDTClientCacheCleanInterval)
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
	rawValue := os.Getenv(KubemonEnableOperand)
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
	return parseDuration(ctx, WebhookCertsRequeueAfterEnvVar, defaultWebhookCertsRequeueAfter, minWebhookCertsRequeueAfter, maxWebhookCertsRequeueAfter)
}

func GetWebhookCertsRenewalThreshold(ctx context.Context) time.Duration {
	return parseDuration(ctx, WebhookCertsRenewalThresholdEnvVar, defaultWebhookCertsRenewalThreshold, minWebhookCertsRenewalThreshold, maxWebhookCertsRenewalThreshold)
}

func GetWebhookCertsServerDuration(ctx context.Context) time.Duration {
	return parseDuration(ctx, WebhookCertsServerDurationEnvVar, defaultWebhookCertsServerDuration, minWebhookCertsServerDuration, maxWebhookCertsServerDuration)
}

func GetWebhookCertsRootDuration(ctx context.Context) time.Duration {
	return parseDuration(ctx, WebhookCertsRootDurationEnvVar, defaultWebhookCertsRootDuration, minWebhookCertsRootDuration, maxWebhookCertsRootDuration)
}

func parseDuration(ctx context.Context, envVar string, defaultValue, minValue, maxValue time.Duration) time.Duration {
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
