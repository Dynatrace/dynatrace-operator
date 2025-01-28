package prioritymap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAppend(t *testing.T) {
	t.Run("with single env vars", func(t *testing.T) {
		argMap := New()

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
		}, WithPriority(MediumPriority))

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
	})
	t.Run("with sliced env vars", func(t *testing.T) {
		argMap := New()

		valueSource := &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}
		Append(argMap,
			[]corev1.EnvVar{
				{
					Name:      "DT_NODE_NAME",
					ValueFrom: valueSource,
				},
				{
					Name:  "DT_CLUSTER_ID",
					Value: "abcdef-ghijkl",
				},
			},
		)

		expectedEnvVars := []corev1.EnvVar{
			{
				Name:  "DT_CLUSTER_ID",
				Value: "abcdef-ghijkl",
			},
			{
				Name:      "DT_NODE_NAME",
				ValueFrom: valueSource,
			},
		}
		assert.Equal(t, expectedEnvVars, argMap.AsEnvVars())
	})
	t.Run("with string map", func(t *testing.T) {
		argMap := New()

		Append(argMap,
			map[string]any{
				"DT_CLUSTER_ID": "abcdef-ghijkl",
			},
		)

		expectedEnvVars := []corev1.EnvVar{
			{
				Name:  "DT_CLUSTER_ID",
				Value: "abcdef-ghijkl",
			},
		}
		assert.Equal(t, expectedEnvVars, argMap.AsEnvVars())
	})
}

func TestWithArguments(t *testing.T) {
	argMap := New()
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

	argMap := New(WithSeparator("="))
	argMap.Append("--set-proxy", "192.168.1.1")
	Append(argMap, args, WithPriority(HighPriority))

	assert.Equal(t, expectedArgs, argMap.AsKeyValueStrings())
}

func TestDuplicateArguments(t *testing.T) {
	defaultArgs := []string{
		"--set-proxy=127.0.0.1",
		"--set-host-id-source=auto",
		"--set-server=localhost",
		"--set-host-property=prop1",
		"--set-host-property=prop2",
		"--set-host-property=prop3",
		"--set-host-property=prop3",
		"--set-host-property=prop3",
	}

	customArgs := []string{
		"--set-host-id-source=fqdn",
		"--set-server=foobar.com",
		"--set-host-property=custom-prop1",
		"--set-host-property=custom-prop2",
		"--set-host-property=custom-prop3",
		"--set-host-property=custom-prop3",
		"--set-host-property=custom-prop3",
	}

	var tests = []struct {
		title          string
		expectedArgs   []string
		defaultArgPrio int
		customArgsPrio int
		mapOptions     []Option
	}{
		{
			title: "Enter avoided duplicates with higher prio",
			expectedArgs: []string{
				"--set-host-id-source=fqdn",
				"--set-host-property=prop1",
				"--set-host-property=prop2",
				"--set-host-property=prop3",
				"--set-host-property=custom-prop1",
				"--set-host-property=custom-prop2",
				"--set-host-property=custom-prop3",
				"--set-proxy=127.0.0.1",
				"--set-server=foobar.com",
			},
			defaultArgPrio: DefaultPriority,
			customArgsPrio: HighPriority,
			mapOptions: []Option{
				WithSeparator("="),
				WithAllowDuplicates(),
				WithAvoidDuplicatesFor("--set-host-id-source"),
				WithAvoidDuplicatesFor("--set-server"),
			},
		},
		{
			title: "Enter avoided duplicates with lower prio",
			expectedArgs: []string{
				"--set-host-id-source=auto",
				"--set-host-property=custom-prop1",
				"--set-host-property=custom-prop2",
				"--set-host-property=custom-prop3",
				"--set-host-property=prop1",
				"--set-host-property=prop2",
				"--set-host-property=prop3",
				"--set-proxy=127.0.0.1",
				"--set-server=localhost",
			},
			defaultArgPrio: HighPriority,
			customArgsPrio: DefaultPriority,
			mapOptions: []Option{
				WithSeparator("="),
				WithAllowDuplicates(),
				WithAvoidDuplicatesFor("--set-host-id-source"),
				WithAvoidDuplicatesFor("--set-server"),
			},
		},
		{
			title: "Enter avoided duplicates with higher prio",
			expectedArgs: []string{
				"--set-host-id-source=fqdn",
				"--set-host-property=prop1",
				"--set-host-property=prop2",
				"--set-host-property=prop3",
				"--set-host-property=custom-prop1",
				"--set-host-property=custom-prop2",
				"--set-host-property=custom-prop3",
				"--set-proxy=127.0.0.1",
				"--set-server=foobar.com",
			},
			defaultArgPrio: DefaultPriority,
			customArgsPrio: HighPriority,
			mapOptions: []Option{
				WithSeparator("="),
				WithAvoidDuplicates(),
				WithAllowDuplicatesFor("--set-host-property"),
			},
		},
		{
			title: "Enter avoided duplicates with lower prio",
			expectedArgs: []string{
				"--set-host-id-source=auto",
				"--set-host-property=custom-prop1",
				"--set-host-property=custom-prop2",
				"--set-host-property=custom-prop3",
				"--set-host-property=prop1",
				"--set-host-property=prop2",
				"--set-host-property=prop3",
				"--set-proxy=127.0.0.1",
				"--set-server=localhost",
			},
			defaultArgPrio: HighPriority,
			customArgsPrio: DefaultPriority,
			mapOptions: []Option{
				WithSeparator("="),
				WithAvoidDuplicates(),
				WithAllowDuplicatesFor("--set-host-property"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			argMap := New(tt.mapOptions...)

			Append(argMap, defaultArgs, WithPriority(tt.defaultArgPrio))
			Append(argMap, customArgs, WithPriority(tt.customArgsPrio))

			assert.Equal(t, tt.expectedArgs, argMap.AsKeyValueStrings())
		})
	}
}
