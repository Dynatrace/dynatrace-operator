package csiprovisioner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeLine(t *testing.T) {
	testConfMap := map[string]map[string]string{
		"general": {
			"prop1": "val1",
		},
	}
	t.Run(`key not in map`, func(t *testing.T) {
		testLine := "prop2 val2"
		merged := mergeLine(testLine, "general", testConfMap)
		assert.Equal(t, "prop2 val2", merged)
	})
	t.Run(`key in map`, func(t *testing.T) {
		testLine := "prop1 val2"
		merged := mergeLine(testLine, "general", testConfMap)
		assert.Equal(t, "prop1 val1", merged)
	})
}

func TestAddLeftoversForSection(t *testing.T) {
	t.Run(`multiple sections`, func(t *testing.T) {
		testConfMap := map[string]map[string]string{
			"general": {
				"prop1": "val1",
				"prop2": "val2",
			},
			"other": {
				"prop1": "val1",
				"prop2": "val2",
			},
		}
		leftovers := addLeftoversForSection("general", testConfMap)
		assert.Len(t, testConfMap, 1)
		assert.Len(t, leftovers, 2)
		assert.Equal(t, []string{"prop1 val1", "prop2 val2"}, leftovers)
	})
	t.Run(`1 section`, func(t *testing.T) {
		testConfMap := map[string]map[string]string{
			"general": {
				"prop1": "val1",
				"prop2": "val2",
			},
		}
		leftovers := addLeftoversForSection("general", testConfMap)
		assert.Len(t, testConfMap, 0)
		assert.Len(t, leftovers, 2)
		assert.Equal(t, []string{"prop1 val1", "prop2 val2"}, leftovers)
	})
	t.Run(`0 section`, func(t *testing.T) {
		testConfMap := map[string]map[string]string{}
		leftovers := addLeftoversForSection("general", testConfMap)
		assert.Len(t, testConfMap, 0)
		assert.Len(t, leftovers, 0)
	})
}
