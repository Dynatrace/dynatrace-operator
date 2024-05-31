package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractVersion(t *testing.T) {
	t.Run("ExtractSemanticVersion", func(t *testing.T) {
		version, err := ExtractSemanticVersion("1.203.0.20200908-220956")
		require.NoError(t, err)
		assert.NotNil(t, version)

		version, err = ExtractSemanticVersion("2.003.0.20200908-220956")
		require.NoError(t, err)
		assert.NotNil(t, version)

		version, err = ExtractSemanticVersion("1.003.5.20200908-220956")
		require.NoError(t, err)
		assert.NotNil(t, version)
	})
	t.Run("ExtractSemanticVersion fails on malformed version", func(t *testing.T) {
		version, err := ExtractSemanticVersion("1.203")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractSemanticVersion("2.003.x.20200908-220956")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractSemanticVersion("")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractSemanticVersion("abc")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractSemanticVersion("a.bcd.e")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractSemanticVersion("asdadasdsd.asd1.2.3")
		assertIsDefaultVersionInfo(t, version, err)
	})
}

func assertIsDefaultVersionInfo(t *testing.T, version SemanticVersion, err error) {
	require.Error(t, err)
	assert.NotNil(t, version)
	assert.Equal(t, SemanticVersion{
		major:     0,
		minor:     0,
		release:   0,
		timestamp: "",
	}, version)
}

func TestCompareClusterVersion(t *testing.T) {
	makeVer := func(major, minor, release int, timestamp string) SemanticVersion {
		return SemanticVersion{
			major:     major,
			minor:     minor,
			release:   release,
			timestamp: timestamp,
		}
	}

	t.Run("CompareSemanticVersions a == b", func(t *testing.T) {
		assert.Equal(t, 0, CompareSemanticVersions(makeVer(1, 200, 0, ""), makeVer(1, 200, 0, "")))
	})

	t.Run("CompareSemanticVersions a < b", func(t *testing.T) {
		assert.Negative(t, CompareSemanticVersions(makeVer(1, 0, 0, ""), makeVer(1, 200, 0, "")))
		assert.Negative(t, CompareSemanticVersions(makeVer(0, 0, 0, ""), makeVer(0, 2000, 3000, "")))
		assert.Negative(t, CompareSemanticVersions(makeVer(1, 200, 0, ""), makeVer(1, 200, 1, "")))
		assert.Negative(t, CompareSemanticVersions(makeVer(1, 200, 0, "0"), makeVer(1, 200, 1, "1")))
	})

	t.Run("CompareSemanticVersions a > b", func(t *testing.T) {
		assert.Positive(t, CompareSemanticVersions(makeVer(1, 200, 0, ""), makeVer(1, 100, 0, "")))
		assert.Positive(t, CompareSemanticVersions(makeVer(2, 0, 0, ""), makeVer(1, 100, 0, "")))
		assert.Positive(t, CompareSemanticVersions(makeVer(1, 201, 0, ""), makeVer(1, 100, 0, "")))
		assert.Positive(t, CompareSemanticVersions(makeVer(1, 0, 0, ""), makeVer(0, 0, 20000, "")))
		assert.Positive(t, CompareSemanticVersions(makeVer(1, 0, 0, "1"), makeVer(1, 0, 0, "0")))
	})
}

func TestNeedsUpgradeRaw(t *testing.T) {
	res, err := IsDowngrade("1.203.0.20200908-220956", "1.203.0.20210908-220956") // Upgrade
	require.NoError(t, err)
	assert.False(t, res)

	res, err = IsDowngrade("1.203.1.20210908-220956", "1.203.0.20200908-220956") // Downgrade
	require.NoError(t, err)
	assert.True(t, res)

	res, err = IsDowngrade("1.203.0.20200908-220956", "1.203.0.20200908-220956") // Same versions
	require.NoError(t, err)
	assert.False(t, res)
}
