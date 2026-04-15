package version

import (
	"strconv"
	"sync"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var log = logd.Get().WithName("k8sversion")

const refreshInterval = 5 * time.Minute

type versionInfoCache struct {
	// Using Mutex over RWMutex is a ~5% perf decrease according to the benchmark, but it makes the logic a lot simpler.
	mutex        sync.Mutex
	lastCheck    time.Time
	minorVersion int
	// Should only be set for testing
	disableLookup bool
}

var versionInfo = new(versionInfoCache)

// DisableCacheForTest disables the global version cache for testing and configures a static minor version that GetMinorVersion will return.
// The returned function can be used to undo this change. Calling this function multiple times with different inputs will update the minorVersion,
// but only the first undo function will revert changes.
//
// Example:
//
//	func TestFoo(t *testing.T) {
//		t.Cleanup(version.DisableCacheForTest(34))
//		...
//		// Subsequent calls can ignore the return value
//		DisableCacheForTest(35)
//	}
func DisableCacheForTest(minorVersion int) func() {
	if versionInfo.disableLookup {
		versionInfo.minorVersion = minorVersion
		// Set the version, but only undo the first time
		return func() {}
	}

	prevMinorVersion := versionInfo.minorVersion
	versionInfo.minorVersion = minorVersion
	versionInfo.disableLookup = true

	return func() {
		versionInfo.minorVersion = prevMinorVersion
		versionInfo.disableLookup = false
	}
}

// GetMinorVersion looks up the Kubernetes minor version from the cluster.
// The version is cached and refreshed every 5 minutes, if requested.
//
// This function is safe to be called concurrently.
func GetMinorVersion() int {
	return versionInfo.getMinorVersion()
}

func (c *versionInfoCache) getMinorVersion() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.refreshMinorVersion(); err != nil {
		log.Error(err, "kubernetes version lookup failed")
	}

	return c.minorVersion
}

var (
	getConfig = config.GetConfig
	newClient = newDiscoveryClientForConfig
)

func newDiscoveryClientForConfig(c *rest.Config) (discovery.ServerVersionInterface, error) {
	return discovery.NewDiscoveryClientForConfig(c)
}

func (c *versionInfoCache) refreshMinorVersion() error {
	if c.disableLookup || !c.shouldRefresh() {
		return nil
	}

	// Set the timestamp early to ensure we only run this once ever time slot.
	// Any errors that occur will likely not go away in the timespan of a mutex lock/unlock,
	// so don't spam the logs with them.
	// IMPORTANT: All errors should include a stacktrace to improve visibility.
	c.lastCheck = time.Now()

	cfg, err := getConfig()
	if err != nil {
		return errors.Wrap(err, "load kubeconfig")
	}

	client, err := newClient(cfg)
	if err != nil {
		return errors.Wrap(err, "build discovery client")
	}

	info, err := GetServerVersion(client)
	if err != nil {
		return errors.Wrap(err, "get kubernetes server version")
	}

	minor, err := strconv.Atoi(info.Minor)
	if err != nil {
		return errors.Wrap(err, "invalid kubernetes minor version")
	}

	log.Debug("cached kubernetes server version", "minorVersion", info.Minor)

	c.minorVersion = minor

	return nil
}

func (c *versionInfoCache) shouldRefresh() bool {
	return c.lastCheck.IsZero() || time.Since(c.lastCheck) >= refreshInterval
}

// Reset cache by re-initializing it to the zero value. Do not call this concurrently or when the cache is under lock!
func resetCache() {
	versionInfo = new(versionInfoCache)
}
