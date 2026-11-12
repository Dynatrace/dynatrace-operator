// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNameInvalid(t *testing.T) {
	clt := fake.NewClient()

	t.Run("valid names", func(t *testing.T) {
		for _, name := range []string{"pm", "my-pm-1", strings.Repeat("a", maxNameLength)} {
			assert.Empty(t, nameInvalid(t.Context(), clt, newPM(name, "ns", nil)), name)
		}
	})

	t.Run("too long name", func(t *testing.T) {
		msg := nameInvalid(t.Context(), clt, newPM(strings.Repeat("a", maxNameLength+1), "ns", nil))

		assert.Contains(t, msg, "length limit")
	})

	t.Run("not a DNS-1035 label", func(t *testing.T) {
		for _, name := range []string{"1pm", "pm-", "Pm", "p.m"} {
			assert.Equal(t, errorNoDNS1035Label(), nameInvalid(t.Context(), clt, newPM(name, "ns", nil)), name)
		}
	})

	t.Run("too long name reports the length, not DNS-1035", func(t *testing.T) {
		msg := nameInvalid(t.Context(), clt, newPM("1"+strings.Repeat("a", maxNameLength), "ns", nil))

		assert.Contains(t, msg, "length limit")
	})
}

func TestValidateRejectsInvalidName(t *testing.T) {
	validator := &Validator{apiReader: fake.NewClient()}
	pm := newPM(strings.Repeat("a", maxNameLength+1), "ns", nil)

	t.Run("ValidateCreate", func(t *testing.T) {
		_, err := validator.ValidateCreate(t.Context(), pm)

		require.Error(t, err)
	})

	t.Run("ValidateUpdate", func(t *testing.T) {
		_, err := validator.ValidateUpdate(t.Context(), pm, pm)

		require.Error(t, err)
	})
}
