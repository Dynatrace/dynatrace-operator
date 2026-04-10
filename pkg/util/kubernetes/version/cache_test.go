package version

import (
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	k8sversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestDisableCacheForTest(t *testing.T) {
	resetCache()

	getConfig = func() (*rest.Config, error) {
		panic("unexpected call")
	}

	newClient = func(*rest.Config) (discovery.ServerVersionInterface, error) {
		panic("unexpected call")
	}

	t.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	actualUndo := DisableCacheForTest(123)

	assert.Equal(t, 123, versionInfo.minorVersion)
	assert.True(t, versionInfo.disableLookup)
	assert.NotPanics(t, func() { GetMinorVersion() })

	nopUndo := DisableCacheForTest(321)
	assert.Equal(t, 321, versionInfo.minorVersion)
	assert.True(t, versionInfo.disableLookup)

	nopUndo()

	assert.Equal(t, 321, versionInfo.minorVersion)
	assert.True(t, versionInfo.disableLookup)
	assert.NotPanics(t, func() { GetMinorVersion() })

	actualUndo()

	assert.Equal(t, 0, versionInfo.minorVersion)
	assert.False(t, versionInfo.disableLookup)
	assert.Panics(t, func() { GetMinorVersion() })
}

func TestGetMinorVersion(t *testing.T) {
	var counter atomic.Int64

	getConfig = func() (*rest.Config, error) {
		counter.Add(1)

		return &rest.Config{}, nil
	}

	newClient = func(*rest.Config) (discovery.ServerVersionInterface, error) {
		fakeClientSet := fakeclientset.NewClientset()
		fakeDiscovery := fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery)
		fakeDiscovery.FakedServerVersion = &k8sversion.Info{
			Major:      "1",
			Minor:      "34",
			GitVersion: "v1.34.4-gke.1047000",
			Platform:   "linux/amd64",
		}

		return fakeDiscovery, nil
	}

	t.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	resetCache()

	synctest.Test(t, func(t *testing.T) {
		const iterations = 1_000
		results := make([]int, iterations)
		for i := range iterations {
			go func() {
				results[i] = GetMinorVersion()
			}()
		}

		synctest.Wait()

		assert.Equal(t, slices.Repeat([]int{34}, iterations), results)
		assert.Equal(t, int64(1), counter.Load())

		time.Sleep(refreshInterval + 1)
		_ = GetMinorVersion()
		assert.Equal(t, int64(2), counter.Load())
	})
}

func TestGetMinorVersionOnError(t *testing.T) {
	var counter atomic.Int64

	getConfig = func() (*rest.Config, error) {
		counter.Add(1)

		return &rest.Config{}, nil
	}

	newClient = func(*rest.Config) (discovery.ServerVersionInterface, error) {
		fakeClientSet := fakeclientset.NewClientset()
		fakeDiscovery := fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery)
		fakeDiscovery.FakedServerVersion = &k8sversion.Info{
			Major:      "1",
			Minor:      "invalid",
			GitVersion: "v1.34.4-gke.1047000",
			Platform:   "linux/amd64",
		}

		return fakeDiscovery, nil
	}

	t.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	resetCache()

	synctest.Test(t, func(t *testing.T) {
		const iterations = 10
		for range iterations {
			go func() {
				GetMinorVersion()
			}()
		}

		synctest.Wait()

		assert.Equal(t, int64(1), counter.Load())
	})
}

func BenchmarkGetMinorVersionUncached(b *testing.B) {
	getConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}

	newClient = func(*rest.Config) (discovery.ServerVersionInterface, error) {
		fakeClientSet := fakeclientset.NewClientset()
		fakeDiscovery := fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery)
		fakeDiscovery.FakedServerVersion = &k8sversion.Info{
			Major:      "1",
			Minor:      "34",
			GitVersion: "v1.34.4-gke.1047000",
			Platform:   "linux/amd64",
		}

		return fakeDiscovery, nil
	}

	b.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	resetCache()

	for b.Loop() {
		if v := GetMinorVersion(); v != 34 {
			b.Fatalf("got unexpected minor version: %d", v)
		}
		// reset clock
		versionInfo.lastCheck = time.Time{}
	}
}

func BenchmarkGetMinorVersionCached(b *testing.B) {
	getConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}

	newClient = func(*rest.Config) (discovery.ServerVersionInterface, error) {
		fakeClientSet := fakeclientset.NewClientset()
		fakeDiscovery := fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery)
		fakeDiscovery.FakedServerVersion = &k8sversion.Info{
			Major:      "1",
			Minor:      "34",
			GitVersion: "v1.34.4-gke.1047000",
			Platform:   "linux/amd64",
		}

		return fakeDiscovery, nil
	}

	b.Cleanup(func() {
		getConfig = config.GetConfig
		newClient = newDiscoveryClientForConfig
	})

	resetCache()

	// Initialize cache
	_ = GetMinorVersion()

	for b.Loop() {
		if v := GetMinorVersion(); v != 34 {
			b.Fatalf("got unexpected minor version: %d", v)
		}
	}
}
