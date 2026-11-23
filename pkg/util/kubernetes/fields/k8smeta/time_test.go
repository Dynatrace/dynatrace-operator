// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8smeta

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTimeoutReached(t *testing.T) {
	const timeout = 15 * time.Minute

	t.Run("nil => reached", func(t *testing.T) {
		assert.True(t, TimeoutReached(nil, timeout))
	})

	t.Run("zero => reached", func(t *testing.T) {
		assert.True(t, TimeoutReached(&metav1.Time{}, timeout))
	})

	t.Run("reached exactly at timeout", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			last := metav1.Now()

			assert.False(t, TimeoutReached(&last, timeout))

			synctest.Sleep(timeout - time.Nanosecond)

			assert.False(t, TimeoutReached(&last, timeout))

			synctest.Sleep(time.Nanosecond)

			assert.True(t, TimeoutReached(&last, timeout))

			synctest.Sleep(time.Hour)

			assert.True(t, TimeoutReached(&last, timeout))
		})
	})

	t.Run("zero timeout => reached immediately", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			last := metav1.Now()

			assert.True(t, TimeoutReached(&last, 0))
		})
	})
}
