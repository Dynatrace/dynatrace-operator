package csiprovisioner

import "testing"

func TestGetInstaller(t *testing.T) {
	t.Run("version set => zip installer", func(t *testing.T) {

	})

	t.Run("image set => image installer", func(t *testing.T) {

	})

	t.Run("nothing set => error (shouldn't be possible in real life)", func(t *testing.T) {

	})
}

func TestGetTargetDir(t *testing.T) {
	t.Run("version set => folder is the version", func(t *testing.T) {

	})

	t.Run("image set => folder is the base64 of the imageURI", func(t *testing.T) {

	})

	t.Run("nothing set => folder is called `unknown` (shouldn't be possible in real life)", func(t *testing.T) {

	})
}
