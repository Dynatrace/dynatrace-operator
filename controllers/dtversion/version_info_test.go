package dtversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractVersion(t *testing.T) {
	t.Run("ExtractVersion", func(t *testing.T) {
		version, err := ExtractVersion("1.203.0.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)

		version, err = ExtractVersion("2.003.0.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)

		version, err = ExtractVersion("1.003.5.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)
	})
	t.Run("ExtractVersion fails on malformed version", func(t *testing.T) {
		version, err := ExtractVersion("1.203")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractVersion("2.003.x.20200908-220956")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractVersion("")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractVersion("abc")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractVersion("a.bcd.e")
		assertIsDefaultVersionInfo(t, version, err)

		version, err = ExtractVersion("asdadasdsd.asd1.2.3")
		assertIsDefaultVersionInfo(t, version, err)
	})
}

func assertIsDefaultVersionInfo(t *testing.T, version VersionInfo, err error) {
	assert.Error(t, err)
	assert.NotNil(t, version)
	assert.Equal(t, VersionInfo{
		major:     0,
		minor:     0,
		release:   0,
		timestamp: "",
	}, version)
}

func TestCompareClusterVersion(t *testing.T) {
	makeVer := func(major, minor, release int, timestamp string) VersionInfo {
		return VersionInfo{
			major:     major,
			minor:     minor,
			release:   release,
			timestamp: timestamp,
		}
	}

	t.Run("CompareVersionInfo a == b", func(t *testing.T) {
		assert.Equal(t, 0, CompareVersionInfo(makeVer(1, 200, 0, ""), makeVer(1, 200, 0, "")))
	})

	t.Run("CompareVersionInfo a < b", func(t *testing.T) {
		assert.Less(t, CompareVersionInfo(makeVer(1, 0, 0, ""), makeVer(1, 200, 0, "")), 0)
		assert.Less(t, CompareVersionInfo(makeVer(0, 0, 0, ""), makeVer(0, 2000, 3000, "")), 0)
		assert.Less(t, CompareVersionInfo(makeVer(1, 200, 0, ""), makeVer(1, 200, 1, "")), 0)
		assert.Less(t, CompareVersionInfo(makeVer(1, 200, 0, "0"), makeVer(1, 200, 1, "1")), 0)
	})

	t.Run("CompareVersionInfo a > b", func(t *testing.T) {
		assert.Greater(t, CompareVersionInfo(makeVer(1, 200, 0, ""), makeVer(1, 100, 0, "")), 0)
		assert.Greater(t, CompareVersionInfo(makeVer(2, 0, 0, ""), makeVer(1, 100, 0, "")), 0)
		assert.Greater(t, CompareVersionInfo(makeVer(1, 201, 0, ""), makeVer(1, 100, 0, "")), 0)
		assert.Greater(t, CompareVersionInfo(makeVer(1, 0, 0, ""), makeVer(0, 0, 20000, "")), 0)
		assert.Greater(t, CompareVersionInfo(makeVer(1, 0, 0, "1"), makeVer(1, 0, 0, "0")), 0)
	})
}

func TestNeedsUpgradeRaw(t *testing.T) {
	res, err := NeedsUpgradeRaw("1.203.0.20200908-220956", "1.203.0.20210908-220956") // Upgrade
	assert.True(t, res)
	assert.NoError(t, err)

	_, err = NeedsUpgradeRaw("1.203.1.20210908-220956", "1.203.0.20200908-220956") // Downgrade
	assert.Error(t, err)

	res, err = NeedsUpgradeRaw("1.203.0.20200908-220956", "1.203.0.20200908-220956") // Same versions
	assert.False(t, res)
	assert.NoError(t, err)
}
