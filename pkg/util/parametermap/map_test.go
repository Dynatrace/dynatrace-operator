package parametermap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestWithEnvVars(t *testing.T) {
	argMap := NewMap()

	// value source
	valueSource := &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}
	Append(argMap, corev1.EnvVar{
		Name:      "DT_NODE_NAME",
		ValueFrom: valueSource,
	})

	// value
	Append(argMap, corev1.EnvVar{
		Name:  "DT_CLUSTER_ID",
		Value: "abcdef-ghijkl",
	})

	// value
	Append(argMap, corev1.EnvVar{
		Name:      "DT_TENANT",
		ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.tenant"}},
	})

	// value
	Append(argMap, corev1.EnvVar{
		Name:  "DT_TENANT",
		Value: "abc12345",
	}, WithPriority(2))

	// strings
	Append(argMap, []string{
		"TESTVAR1=herbert",
		"VARTEST1=waltraud",
	})

	expectedEnvVars := []corev1.EnvVar{
		{
			Name:  "DT_CLUSTER_ID",
			Value: "abcdef-ghijkl",
		},
		{
			Name:      "DT_NODE_NAME",
			ValueFrom: valueSource,
		},
		{
			Name:  "DT_TENANT",
			Value: "abc12345",
		},
		{
			Name:  "TESTVAR1",
			Value: "herbert",
		},
		{
			Name:  "VARTEST1",
			Value: "waltraud",
		},
	}
	assert.Equal(t, expectedEnvVars, argMap.AsEnvVars())
}

func TestWithArguments(t *testing.T) {
	argMap := NewMap()
	argMap.Append("--proxy=", "127.0.0.1", WithSeparator(DefaultSeparator))
	argMap.Append("--tenant", "$(DT_TENANT)", WithSeparator(DefaultSeparator))
	argMap.Append("--no-foobar", "")
	argMap.Append("--hubert=", "", WithSeparator(DefaultSeparator))
	argMap.Append("karlheinz=", "", WithSeparator(DefaultSeparator))
	argMap.Append("", "", WithSeparator(DefaultSeparator))

	expectedArgs := []string{
		"--hubert=",
		"--no-foobar",
		"--proxy=127.0.0.1",
		"--tenant=$(DT_TENANT)",
		"karlheinz=",
	}

	args := argMap.AsKeyValueStrings()
	assert.Equal(t, expectedArgs, args)
}

func TestArgumentSlice(t *testing.T) {
	args := []string{
		"--set-proxy=127.0.0.1",
		"tenant=abcd1345",
		"-foobar",
		"=17",
		"",
		"=",
		"set-proxy=1.2.3.4",
	}
	expectedArgs := []string{
		"--set-proxy=127.0.0.1",
		"-foobar",
		"set-proxy=1.2.3.4",
		"tenant=abcd1345",
	}

	argMap := NewMap(WithSeparator("="))
	argMap.Append("--set-proxy", "192.168.1.1")
	Append(argMap, args, WithPriority(2))

	assert.Equal(t, expectedArgs, argMap.AsKeyValueStrings())
}
