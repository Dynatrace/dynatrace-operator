package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractVersion(t *testing.T) {
	t.Run("extractVersion", func(t *testing.T) {
		version, err := extractVersion("1.203.0.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)

		version, err = extractVersion("2.003.0.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)

		version, err = extractVersion("1.003.5.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)
	})
	t.Run("extractVersion fails on malformed version", func(t *testing.T) {
		version, err := extractVersion("1.203")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = extractVersion("2.003.x.20200908-220956")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = extractVersion("")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = extractVersion("abc")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = extractVersion("a.bcd.e")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = extractVersion("asdadasdsd.asd1.2.3")
		assertIsDefaultVersionInfo(t, version, err)
	})
}

func assertIsDefaultVersionInfo(t *testing.T, version versionInfo, err error) {
	assert.Error(t, err)
	assert.NotNil(t, version)
	assert.Equal(t, versionInfo{
		major:   0,
		minor:   0,
		release: 0,
	}, version)
}

func TestCompareClusterVersion(t *testing.T) {
	t.Run("compareVersionInfo a == b", func(t *testing.T) {
		a := versionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		b := versionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		comparison := compareVersionInfo(a, b)
		assert.Equal(t, 0, comparison)
	})
	t.Run("compareVersionInfo a < b", func(t *testing.T) {
		a := versionInfo{
			major:   1,
			minor:   0,
			release: 0,
		}
		b := versionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		comparison := compareVersionInfo(a, b)
		assert.Less(t, comparison, 0)

		a = versionInfo{
			major:   0,
			minor:   0,
			release: 0,
		}
		comparison = compareVersionInfo(a, b)
		assert.Less(t, comparison, 0)

		a = versionInfo{
			major:   0,
			minor:   2000,
			release: 3000,
		}
		comparison = compareVersionInfo(a, b)
		assert.Less(t, comparison, 0)

		a = versionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		b = versionInfo{
			major:   1,
			minor:   200,
			release: 1,
		}

		comparison = compareVersionInfo(a, b)
		assert.Less(t, comparison, 0)

	})
	t.Run("compareVersionInfo a > b", func(t *testing.T) {
		a := versionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		b := versionInfo{
			major:   1,
			minor:   100,
			release: 0,
		}
		comparison := compareVersionInfo(a, b)
		assert.Greater(t, comparison, 0)

		a = versionInfo{
			major:   2,
			minor:   0,
			release: 0,
		}
		comparison = compareVersionInfo(a, b)
		assert.Greater(t, comparison, 0)

		a = versionInfo{
			major:   1,
			minor:   201,
			release: 0,
		}
		comparison = compareVersionInfo(a, b)
		assert.Greater(t, comparison, 0)

		a = versionInfo{
			major:   1,
			minor:   0,
			release: 0,
		}
		b = versionInfo{
			major:   0,
			minor:   0,
			release: 20000,
		}

		comparison = compareVersionInfo(a, b)
		assert.Greater(t, comparison, 0)
	})
}
