// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sobject

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CompositeWaitConditions(t *testing.T) {
	neverDone := func(context.Context) (done bool, err error) {
		return false, nil
	}

	alwaysErr := func(context.Context) (done bool, err error) {
		return false, errors.New("error")
	}

	alwaysOk := func(context.Context) (done bool, err error) {
		return true, nil
	}

	t.Run("empty", func(t *testing.T) {
		done, err := CompositeWaitConditions()(context.Background())
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("passthrough done", func(t *testing.T) {
		done, err := CompositeWaitConditions(alwaysOk)(context.Background())
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("passthrough not done", func(t *testing.T) {
		done, err := CompositeWaitConditions(neverDone)(context.Background())
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("passthrough error", func(t *testing.T) {
		done, err := CompositeWaitConditions(alwaysErr)(context.Background())
		require.Error(t, err)
		assert.False(t, done)
	})

	t.Run("error priority on done", func(t *testing.T) {
		done, err := CompositeWaitConditions(alwaysOk, neverDone, alwaysErr)(context.Background())
		require.Error(t, err)
		assert.False(t, done)
	})

	t.Run("any not done is not done", func(t *testing.T) {
		done, err := CompositeWaitConditions(alwaysOk, neverDone)(context.Background())
		require.NoError(t, err)
		assert.False(t, done)
	})

}
