package dtclient

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodProcessModuleConfigResponse = `
{
	"revision": 1,
	"properties": [{
			"section": "general",
			"key": "field",
			"value": "test"
		},
		{
			"section": "test",
			"key": "a",
			"value": "b"
		}

	]
}
`

func TestCreateProcessModuleConfigRequest(t *testing.T) {
	dc := &dynatraceClient{
		paasToken: "token123",
	}
	require.NotNil(t, dc)

	req, err := dc.createProcessModuleConfigRequest(0)
	require.Nil(t, err)
	assert.Equal(t, "0", req.URL.Query().Get("revision"))
	assert.Contains(t, req.Header.Get("Authorization"), dc.paasToken)
}

func TestSpecialProcessModuleConfigRequestStatus(t *testing.T) {
	dc := &dynatraceClient{}
	require.NotNil(t, dc)

	assert.True(t, dc.checkProcessModuleConfigRequestStatus(nil))
	assert.True(t, dc.checkProcessModuleConfigRequestStatus(&http.Response{StatusCode: http.StatusNotModified}))
	assert.True(t, dc.checkProcessModuleConfigRequestStatus(&http.Response{StatusCode: http.StatusNotFound}))
	assert.False(t, dc.checkProcessModuleConfigRequestStatus(&http.Response{StatusCode: http.StatusOK}))
	assert.False(t, dc.checkProcessModuleConfigRequestStatus(&http.Response{StatusCode: http.StatusInternalServerError}))
}

func TestReadResponseForProcessModuleConfig(t *testing.T) {
	dc := &dynatraceClient{}
	require.NotNil(t, dc)

	processConfig, err := dc.readResponseForProcessModuleConfig([]byte(goodProcessModuleConfigResponse))
	require.Nil(t, err)
	assert.Equal(t, uint(1), processConfig.Revision)
	require.Len(t, processConfig.Properties, 2)
	assert.Equal(t, "general", processConfig.Properties[0].Section)
	assert.Equal(t, "field", processConfig.Properties[0].Key)
	assert.Equal(t, "test", processConfig.Properties[0].Value)
	assert.Equal(t, "test", processConfig.Properties[1].Section)
	assert.Equal(t, "a", processConfig.Properties[1].Key)
	assert.Equal(t, "b", processConfig.Properties[1].Value)
}

func TestAddHostGroup(t *testing.T) {
	t.Run(`hostGroup, no api`, func(t *testing.T) {
		emptyResponse := ProcessModuleConfig{}
		result := emptyResponse.AddHostGroup("test")
		assert.NotNil(t, result)
		assert.Equal(t, "test", result.ToMap()["general"]["hostGroup"])
	})
	t.Run(`hostGroup, api present`, func(t *testing.T) {
		pmc := ProcessModuleConfig{
			Properties: []ProcessModuleProperty{
				{
					Section: "general",
					Key:     "other",
					Value:   "other",
				},
			},
		}
		result := pmc.AddHostGroup("test")
		assert.NotNil(t, result)
		assert.Len(t, result.ToMap()["general"], 2)
		assert.Equal(t, "test", result.ToMap()["general"]["hostGroup"])
	})
	t.Run(`empty hostGroup`, func(t *testing.T) {
		pmc := ProcessModuleConfig{
			Properties: []ProcessModuleProperty{
				{
					Section: "general",
					Key:     "other",
					Value:   "other",
				},
			},
		}
		result := pmc.AddHostGroup("")
		assert.NotNil(t, result)
		assert.Equal(t, *result, pmc)
	})
	t.Run(`empty hostGroup, remove previous hostgroup`, func(t *testing.T) {
		pmc := ProcessModuleConfig{
			Properties: []ProcessModuleProperty{
				{
					Section: "general",
					Key:     "hostGroup",
					Value:   "other",
				},
			},
		}
		result := pmc.AddHostGroup("")
		assert.NotNil(t, result)
		assert.Len(t, pmc.Properties, 0)
	})
}

const (
	testSection = "test-section"
	testKey     = "test-key"
	testValue   = "test-value"
)

func TestAdd(t *testing.T) {
	t.Run("adds properties", func(t *testing.T) {
		processModuleConfig := &ProcessModuleConfig{}

		for i := 0; i < 10; i++ {
			section := fmt.Sprintf("%s-%d", testSection, i)
			key := fmt.Sprintf("%s-%d", testKey, i)
			value := fmt.Sprintf("%s-%d", testValue, i)

			processModuleConfig.Add(ProcessModuleProperty{
				Section: section,
				Key:     key,
				Value:   value,
			})

			assert.Len(t, processModuleConfig.Properties, i+1)
			assert.Equal(t, section, processModuleConfig.Properties[i].Section)
			assert.Equal(t, key, processModuleConfig.Properties[i].Key)
			assert.Equal(t, value, processModuleConfig.Properties[i].Value)
		}
	})
	t.Run("does not add empty values", func(t *testing.T) {
		processModuleConfig := &ProcessModuleConfig{}
		processModuleConfig.Add(ProcessModuleProperty{
			Section: testSection,
			Key:     testKey,
			Value:   "",
		})

		assert.NotContains(t, processModuleConfig.Properties, ProcessModuleProperty{
			Section: testSection,
			Key:     testKey,
			Value:   "",
		})
	})
	t.Run("does not add same property multiple times", func(t *testing.T) {
		processModuleConfig := &ProcessModuleConfig{}

		for i := 0; i < 10; i++ {
			processModuleConfig.Add(ProcessModuleProperty{
				Section: testSection,
				Key:     testKey,
				Value:   testValue,
			})

			assert.Len(t, processModuleConfig.Properties, 1)
			assert.Equal(t, testSection, processModuleConfig.Properties[0].Section)
			assert.Equal(t, testKey, processModuleConfig.Properties[0].Key)
			assert.Equal(t, testValue, processModuleConfig.Properties[0].Value)
		}
	})
	t.Run("removes property", func(t *testing.T) {
		processModuleConfig := &ProcessModuleConfig{}

		for i := 0; i < 10; i++ {
			section := fmt.Sprintf("%s-%d", testSection, i)
			key := fmt.Sprintf("%s-%d", testKey, i)
			value := fmt.Sprintf("%s-%d", testValue, i)

			processModuleConfig.Add(ProcessModuleProperty{
				Section: section,
				Key:     key,
				Value:   value,
			})
		}

		processModuleConfig.Add(ProcessModuleProperty{
			Section: "test-section-1",
			Key:     "test-key-1",
			Value:   "",
		})

		assert.Len(t, processModuleConfig.Properties, 9)
		assert.NotContains(t, processModuleConfig.Properties, ProcessModuleProperty{
			Section: "test-section-1",
			Key:     "test-key-1",
			Value:   "test-value-1",
		})
		assert.NotContains(t, processModuleConfig.Properties, ProcessModuleProperty{
			Section: "test-section-1",
			Key:     "test-key-1",
			Value:   "",
		})
	})
	t.Run("updates property", func(t *testing.T) {
		processModuleConfig := &ProcessModuleConfig{}

		for i := 0; i < 10; i++ {
			section := fmt.Sprintf("%s-%d", testSection, i)
			key := fmt.Sprintf("%s-%d", testKey, i)
			value := fmt.Sprintf("%s-%d", testValue, i)

			processModuleConfig.Add(ProcessModuleProperty{
				Section: section,
				Key:     key,
				Value:   value,
			})
		}

		processModuleConfig.Add(ProcessModuleProperty{
			Section: "test-section-1",
			Key:     "test-key-1",
			Value:   "new-value",
		})

		assert.Len(t, processModuleConfig.Properties, 10)
		assert.NotContains(t, processModuleConfig.Properties, ProcessModuleProperty{
			Section: "test-section-1",
			Key:     "test-key-1",
			Value:   "test-value-1",
		})
		assert.Contains(t, processModuleConfig.Properties, ProcessModuleProperty{
			Section: "test-section-1",
			Key:     "test-key-1",
			Value:   "new-value",
		})
	})
	t.Run("fixes broken cache", func(t *testing.T) {
		processModuleConfig := &ProcessModuleConfig{}

		for i := 0; i < 10; i++ {
			section := fmt.Sprintf("%s-%d", testSection, i)
			key := fmt.Sprintf("%s-%d", testKey, i)
			value := fmt.Sprintf("%s-%d", testValue, i)

			processModuleConfig.Add(ProcessModuleProperty{
				Section: section,
				Key:     key,
				Value:   value,
			})
		}
		for i := 0; i < 10; i++ {
			processModuleConfig.Properties = append(processModuleConfig.Properties, ProcessModuleProperty{
				Section: testSection,
				Key:     testKey,
				Value:   testValue,
			})
		}

		require.Len(t, processModuleConfig.Properties, 20)

		processModuleConfig.Add(ProcessModuleProperty{
			Section: testSection,
			Key:     testKey,
			Value:   "new-value",
		})

		assert.Len(t, processModuleConfig.Properties, 11)
		assert.Contains(t, processModuleConfig.Properties, ProcessModuleProperty{
			Section: testSection,
			Key:     testKey,
			Value:   "new-value",
		})
		assert.NotContains(t, processModuleConfig.Properties, ProcessModuleProperty{
			Section: testSection,
			Key:     testKey,
			Value:   testValue,
		})
	})
}
