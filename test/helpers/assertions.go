// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// AssertGolden compares obj, marshaled to YAML, against expectedPath. The fake
// client is the only source of non-determinism (it stamps resourceVersion), so
// that field is cleared before comparing.
func AssertGolden(t *testing.T, expectedPath string, obj client.Object) {
	t.Helper()

	obj.SetResourceVersion("")
	kinds, _, _ := scheme.Scheme.ObjectKinds(obj)
	require.Len(t, kinds, 1)
	obj.GetObjectKind().SetGroupVersionKind(kinds[0])

	got, err := yaml.Marshal(obj)
	require.NoError(t, err)

	want, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.YAMLEq(t, string(want), string(got))
}
