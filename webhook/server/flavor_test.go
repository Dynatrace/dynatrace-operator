package server

import (
	"testing"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/webhook"
	"gotest.tools/assert"
)

func TestGetFlavor(t *testing.T) {
	t.Run("TestGetFlavor no value given", func(t *testing.T) {
		assert.Equal(t, "default", getFlavor("", map[string]string{}), "flavor should default to 'default'")
	})
	t.Run("TestGetFlavor wrong value given", func(t *testing.T) {
		assert.Equal(t, "default", getFlavor("wrong value", map[string]string{}), "flavor should default to 'default' if a wrong value is given")
		assert.Equal(t, "default", getFlavor("musl", map[string]string{dtwebhook.AnnotationFlavor: "glibc"}), "flavor should default to 'default' if a wrong value is given")
		assert.Equal(t, "default", getFlavor("musl", map[string]string{dtwebhook.AnnotationFlavor: "wrong value"}), "flavor should default to 'default' if a wrong value is given")
	})
	t.Run("TestGetFlavor libc is set to musl", func(t *testing.T) {
		assert.Equal(t, "musl", getFlavor("musl", map[string]string{}), "flavor should the value as specified")
	})
	t.Run("TestGetFlavor annotation overwrites spec", func(t *testing.T) {
		assert.Equal(t, "musl", getFlavor("glibc", map[string]string{dtwebhook.AnnotationFlavor: "musl"}), "annotation should overwrite given value")
		assert.Equal(t, "default", getFlavor("musl", map[string]string{dtwebhook.AnnotationFlavor: "default"}), "annotation should overwrite given value")
	})
}
