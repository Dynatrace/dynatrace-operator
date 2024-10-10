package exp

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createWithAnnotation(keyValues ...string) FeatureFlags {
	flags := FeatureFlags{
		annotations: map[string]string{},
	}

	for i := 0; i < len(keyValues); i += 2 {
		flags.annotations[keyValues[i]] = keyValues[i+1]
	}

	return flags
}

func createEmpty() FeatureFlags {
	flags := FeatureFlags{
		annotations: map[string]string{},
	}

	return flags
}

func TestCreateWithAnnotation(t *testing.T) {
	f := createWithAnnotation("test", "true")

	assert.Contains(t, f.annotations, "test")
	assert.Equal(t, "true", f.annotations["test"])

	f = createWithAnnotation("other test", "false")

	assert.Contains(t, f.annotations, "other test")
	assert.Equal(t, "false", f.annotations["other test"])
	assert.NotContains(t, f.annotations, "test")

	f = createWithAnnotation("test", "true", "other test", "false")

	assert.Contains(t, f.annotations, "other test")
	assert.Equal(t, "false", f.annotations["other test"])
	assert.Contains(t, f.annotations, "test")
	assert.Equal(t, "true", f.annotations["test"])
}

func testDeprecateDisableAnnotation(t *testing.T,
	newAnnotation string,
	deprecatedAnnotation string,
	propertyFunction func(f FeatureFlags) bool) {
	// New annotation works
	f := createWithAnnotation(newAnnotation, "false")

	assert.True(t, propertyFunction(f))

	f = createWithAnnotation(newAnnotation, "true")

	assert.False(t, propertyFunction(f))

	// Old annotation works
	f = createWithAnnotation(deprecatedAnnotation, "true")

	assert.True(t, propertyFunction(f))

	f = createWithAnnotation(deprecatedAnnotation, "false")

	assert.False(t, propertyFunction(f))

	// New annotation takes precedent
	f = createWithAnnotation(
		newAnnotation, "true",
		deprecatedAnnotation, "true")

	assert.False(t, propertyFunction(f))

	f = createWithAnnotation(
		newAnnotation, "false",
		deprecatedAnnotation, "false")

	assert.True(t, propertyFunction(f))

	// Default is false
	f = createWithAnnotation()

	assert.False(t, propertyFunction(f))
}

func TestDeprecatedDisableAnnotations(t *testing.T) {
	t.Run(ActiveGateUpdatesAnnotation, func(t *testing.T) {
		testDeprecateDisableAnnotation(t,
			ActiveGateUpdatesAnnotation,
			DisableActiveGateUpdatesAnnotation,
			func(f FeatureFlags) bool {
				return f.IsActiveGateUpdatesDisabled()
			})
	})
}

func TestDeprecatedEnableAnnotations(t *testing.T) {
	f := createWithAnnotation(InjectionFailurePolicyAnnotation, "fail")
	assert.Equal(t, "fail", f.GetInjectionFailurePolicy())
}

func TestMaxMountAttempts(t *testing.T) {
	f := createWithAnnotation(
		MaxFailedCsiMountAttemptsAnnotation, "5")

	assert.Equal(t, 5, f.GetMaxFailedCsiMountAttempts())

	f = createWithAnnotation(
		MaxFailedCsiMountAttemptsAnnotation, "3")

	assert.Equal(t, 3, f.GetMaxFailedCsiMountAttempts())

	f = createWithAnnotation()

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, f.GetMaxFailedCsiMountAttempts())

	f = createWithAnnotation(
		MaxFailedCsiMountAttemptsAnnotation, "a")

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, f.GetMaxFailedCsiMountAttempts())

	f = createWithAnnotation(
		MaxFailedCsiMountAttemptsAnnotation, "-5")

	assert.Equal(t, DefaultMaxFailedCsiMountAttempts, f.GetMaxFailedCsiMountAttempts())
}

func TestMaxCSIMountTimeout(t *testing.T) {
	type testCase struct {
		title    string
		input    string
		expected time.Duration
	}

	defaultDuration, err := time.ParseDuration(DefaultMaxCsiMountTimeout)
	require.NoError(t, err)

	tests := []testCase{
		{
			title:    "no annotation -> use default",
			input:    "",
			expected: defaultDuration,
		},
		{
			title:    "incorrect annotation (format) -> use default",
			input:    "5",
			expected: defaultDuration,
		},
		{
			title:    "incorrect annotation (negative) -> use default",
			input:    "-5m",
			expected: defaultDuration,
		},
		{
			title:    "correct annotation -> use value",
			input:    "5m",
			expected: time.Minute * 5,
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			f := createWithAnnotation(MaxCsiMountTimeoutAnnotation, test.input)

			assert.Equal(t, test.expected, f.GetMaxCSIRetryTimeout())
		})
	}
}

func TestMountAttemptsToTimeout(t *testing.T) {
	type testCase struct {
		title    string
		input    int
		expected time.Duration
		delta    float64
	}

	defaultDuration, err := time.ParseDuration(DefaultMaxCsiMountTimeout)
	require.NoError(t, err)

	tests := []testCase{
		{
			title:    "default attempts ~ default duration", // 10 attempts ==> ~8 minutes
			input:    DefaultMaxFailedCsiMountAttempts,
			expected: defaultDuration,
			delta:    float64(time.Minute * 2),
		},

		{
			title:    "1/2 of default attempts ~ NOT 1/2 of default duration (so it is actually exponential)", // 5 attempts ==> ~15 seconds
			input:    DefaultMaxFailedCsiMountAttempts / 2,
			expected: defaultDuration / DefaultMaxFailedCsiMountAttempts / 4,
			delta:    float64(time.Second * 5),
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			duration, err := time.ParseDuration(MountAttemptsToTimeout(test.input))
			require.NoError(t, err)
			assert.InDelta(t, test.expected, duration, test.delta)
		})
	}
}

func TestDynaKube_FeatureIgnoredNamespaces(t *testing.T) {
	object := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
		},
	}

	f := createEmpty()
	ignoredNamespaces := f.getDefaultIgnoredNamespaces(&object)
	namespaceMatches := false

	for _, namespace := range ignoredNamespaces {
		regex, err := regexp.Compile(namespace)

		require.NoError(t, err)

		match := regex.MatchString(object.GetNamespace())

		if match {
			namespaceMatches = true
		}
	}

	assert.True(t, namespaceMatches)
}

func TestDefaultEnabledFeatureFlags(t *testing.T) {
	f := createEmpty()

	assert.True(t, f.IsAutomaticKubernetesApiMonitoringEnabled())
	assert.True(t, f.IsAutomaticInjectionEnabled())
	assert.Equal(t, "silent", f.GetInjectionFailurePolicy())

	assert.False(t, f.IsActiveGateUpdatesDisabled())
	assert.False(t, f.IsLabelVersionDetectionEnabled())
}

func TestInjectionFailurePolicy(t *testing.T) {
	f := createEmpty()

	modes := map[string]string{
		failPhrase:   failPhrase,
		silentPhrase: silentPhrase,
	}
	for configuredMode, expectedMode := range modes {
		t.Run(`injection failure policy: `+configuredMode, func(t *testing.T) {
			f.annotations[InjectionFailurePolicyAnnotation] = configuredMode

			assert.Equal(t, expectedMode, f.GetInjectionFailurePolicy())
		})
	}
}

func TestAgentInitialConnectRetry(t *testing.T) {
	t.Run("default => not set", func(t *testing.T) {
		f := createEmpty()

		initialRetry := f.GetOneAgentInitialConnectRetry(false)
		require.Equal(t, -1, initialRetry)
	})
	t.Run("istio default => set", func(t *testing.T) {
		f := createEmpty()

		initialRetry := f.GetOneAgentInitialConnectRetry(true)
		require.Equal(t, IstioDefaultOneAgentInitialConnectRetry, initialRetry)
	})
	t.Run("istio default can be overruled", func(t *testing.T) {
		f := createEmpty()
		f.annotations[OneAgentInitialConnectRetryAnnotation] = "5"

		initialRetry := f.GetOneAgentInitialConnectRetry(true)
		require.Equal(t, 5, initialRetry)
	})
}
