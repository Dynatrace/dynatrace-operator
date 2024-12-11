package csiprovisioner

import "testing"

func TestReconcile(t *testing.T) {
	t.Run("no dynakube(ie.: delete case) => do nothing, no error", func(t *testing.T) { // TODO: Replace "do nothing" with "run GC"

	})

	t.Run("dynakube doesn't need CSI => only setup base fs, no error, long requeue", func(t *testing.T) {

	})

	t.Run("dynakube doesn't need app-injection => only setup base fs, no error, long requeue", func(t *testing.T) {

	})

	t.Run("dynakube status not ready => only setup base fs, no error, short requeue", func(t *testing.T) {

	})
}

func TestSetupFileSystem(t *testing.T) {
	t.Run("creates necessary folders", func(t *testing.T) {

	})

	t.Run("no error if folders already exist", func(t *testing.T) {

	})
}
