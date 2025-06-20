package installconfig

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	t.Run("empty env -> use fallback", func(t *testing.T) {
		t.Setenv(ModulesJSONEnv, "")

		m := GetModules()
		assert.Equal(t, fallbackModules, m)

		once = sync.Once{} // need to reset it
	})

	t.Run("messy env -> use fallback", func(t *testing.T) {
		t.Setenv(ModulesJSONEnv, "this is not json :(")

		m := GetModules()
		assert.Equal(t, fallbackModules, m)

		once = sync.Once{} // need to reset it
	})

	t.Run("correct env -> set correctly", func(t *testing.T) {
		jsonValue := `
		{
			"csidriver": false,
			"activeGate": true,
			"oneAgent": false,
			"extensions": true,
			"logMonitoring": false,
			"edgeConnect": true,
			"supportability": false,
			"kspm": true
		}`
		expected := Modules{
			CSIDriver:      false,
			ActiveGate:     true,
			OneAgent:       false,
			Extensions:     true,
			LogMonitoring:  false,
			EdgeConnect:    true,
			Supportability: false,
			KSPM:           true,
		}

		t.Setenv(ModulesJSONEnv, jsonValue)

		m := GetModules()
		assert.Equal(t, expected, m)

		once = sync.Once{} // need to reset it
	})

	t.Run("run only once", func(t *testing.T) {
		jsonValue := `
		{
			"csidriver": false,
			"activeGate": true,
			"oneAgent": false,
			"extensions": true,
			"logMonitoring": false,
			"edgeConnect": true,
			"supportability": false,
			"kspm": true
		}`
		expected := Modules{
			CSIDriver:      false,
			ActiveGate:     true,
			OneAgent:       false,
			Extensions:     true,
			LogMonitoring:  false,
			EdgeConnect:    true,
			Supportability: false,
			KSPM:           true,
		}

		t.Setenv(ModulesJSONEnv, jsonValue)

		m := GetModules()

		assert.Equal(t, expected, m)

		t.Setenv(ModulesJSONEnv, "boom")

		m = GetModules()

		assert.Equal(t, expected, m)
	})
}
