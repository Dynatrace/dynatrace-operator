package processmoduleconfig

import (
	"io/ioutil"
	"os"
	"testing"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeLine(t *testing.T) {
	testConfMap := dtclient.ConfMap{
		"general": {
			"prop1": "val1",
		},
	}

	t.Run(`key not map`, func(t *testing.T) {
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
		testConfMap := dtclient.ConfMap{
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
		assert.Contains(t, leftovers, "prop1 val1")
		assert.Contains(t, leftovers, "prop2 val2")
	})
	t.Run(`1 section`, func(t *testing.T) {
		testConfMap := dtclient.ConfMap{
			"general": {
				"prop1": "val1",
			},
		}
		leftovers := addLeftoversForSection("general", testConfMap)
		assert.Empty(t, testConfMap)
		assert.Len(t, leftovers, 1)
		assert.Equal(t, []string{"prop1 val1"}, leftovers)
	})
	t.Run(`0 section`, func(t *testing.T) {
		testConfMap := dtclient.ConfMap{}
		leftovers := addLeftoversForSection("general", testConfMap)
		assert.Empty(t, testConfMap)
		assert.Empty(t, leftovers)
	})
}

func TestAddLeftovers(t *testing.T) {
	t.Run(`multiple sections`, func(t *testing.T) {
		testConfMap := dtclient.ConfMap{
			"general": {
				"prop1": "val1",
			},
		}
		leftovers := addLeftovers(testConfMap)
		assert.Len(t, leftovers, 2)
		assert.Equal(t, []string{"[general]", "prop1 val1"}, leftovers)
	})
}

func TestConfSectionHeader(t *testing.T) {
	header := confSectionHeader("[general]")
	assert.Equal(t, "general", header)
	header = confSectionHeader("general")
	assert.Equal(t, "", header)
	header = confSectionHeader("key val")
	assert.Equal(t, "", header)
	header = confSectionHeader("")
	assert.Equal(t, "", header)
}

func TestStoreConfFile(t *testing.T) {
	memFs := afero.NewMemMapFs()
	expectedOut := `[general]
val key
`

	err := storeFile(memFs, "/dest", 0776, []string{"[general]", "val key"})

	require.NoError(t, err)

	file, _ := memFs.Open("/dest")
	content, _ := ioutil.ReadAll(file)
	assert.Equal(t, expectedOut, string(content))
}

func TestUpdateConfFile(t *testing.T) {
	memFs := afero.NewMemMapFs()
	sourceContent := `[general]
prop1 old

[other]
prop3 old
`
	expected := `[general]
prop1 upd
prop2 new

[other]
prop3 upd
prop4 new

[new]
prop5 new
`
	testConfMap := dtclient.ConfMap{
		"general": {
			"prop1": "upd",
			"prop2": "new",
		},
		"other": {
			"prop3": "upd",
			"prop4": "new",
		},
		"new": {
			"prop5": "new",
		},
	}

	source, _ := memFs.OpenFile("/source", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0776)
	source.WriteString(sourceContent)
	source.Close()

	err := Update(memFs, "/source", "/dest", testConfMap)
	require.NoError(t, err)

	file, _ := memFs.Open("/dest")
	content, _ := ioutil.ReadAll(file)
	assert.Equal(t, expected, string(content))

	source, _ = memFs.Open("/source")
	content, _ = ioutil.ReadAll(source)
	assert.Equal(t, sourceContent, string(content))
}

func TestUpdateConfFileEmptyConfMap(t *testing.T) {
	memFs := afero.NewMemMapFs()
	sourceContent := `[general]
prop1 old

[other]
prop3 old
`
	testConfMap := dtclient.ConfMap{}

	source, _ := memFs.OpenFile("/source", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0776)
	source.WriteString(sourceContent)
	source.Close()

	err := Update(memFs, "/source", "/dest", testConfMap)
	require.NoError(t, err)

	file, _ := memFs.Open("/dest")
	content, _ := ioutil.ReadAll(file)
	assert.Equal(t, sourceContent, string(content))

	source, _ = memFs.Open("/source")
	content, _ = ioutil.ReadAll(source)
	assert.Equal(t, sourceContent, string(content))
}

func TestUpdateConfFileEmptySource(t *testing.T) {
	memFs := afero.NewMemMapFs()
	sourceContent := ``
	testConfMap := dtclient.ConfMap{}

	source, _ := memFs.OpenFile("/source", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0776)
	source.WriteString(sourceContent)
	source.Close()

	err := Update(memFs, "/source", "/dest", testConfMap)
	require.NoError(t, err)

	file, _ := memFs.Open("/dest")
	content, _ := ioutil.ReadAll(file)
	assert.Equal(t, sourceContent, string(content))

	source, _ = memFs.Open("/source")
	content, _ = ioutil.ReadAll(source)
	assert.Equal(t, sourceContent, string(content))
}
