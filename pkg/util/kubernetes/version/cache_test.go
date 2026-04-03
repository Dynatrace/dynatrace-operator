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

	undo := DisableCacheForTest(123)
	assert.True(t, versionInfo.disableLookup)

	synctest.Test(t, func(t *testing.T) {
		const iterations = 1_000
		results := make([]func(), iterations)
		for i := range iterations {
			go func() {
				results[i] = DisableCacheForTest(i)
			}()
		}

		synctest.Wait()

		// The value of minor version is essentially random since we change it concurrently
		assert.NotEqual(t, 123, versionInfo.minorVersion)
		expectMinorVersion := versionInfo.minorVersion

		for idx, f := range results {
			go func() {
				f()
				assert.Truef(t, versionInfo.disableLookup, "undo function %d mutated cache", idx)
				assert.Equal(t, expectMinorVersion, versionInfo.minorVersion, "undo function %d mutated cache", idx)
			}()
		}

		synctest.Wait()
	})

	undo()
	assert.False(t, versionInfo.disableLookup)
	assert.Equal(t, 0, versionInfo.minorVersion)
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
