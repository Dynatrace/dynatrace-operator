package nodes

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/nodes"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	smallBenchmarkConfig = benchmarkConfig{
		NumNodes:     10,
		NumDynakubes: 2,
		NumEntities:  20,
	}
	mediumBenchmarkConfig = benchmarkConfig{
		NumNodes:     50,
		NumDynakubes: 5,
		NumEntities:  100,
	}
	largeBenchmarkConfig = benchmarkConfig{
		NumNodes:     100,
		NumDynakubes: 10,
		NumEntities:  200,
	}
	xlargeBenchmarkConfig = benchmarkConfig{
		NumNodes:     500,
		NumDynakubes: 20,
		NumEntities:  1000,
	}

	// Used for custom benchmarks
	customNumNodes    = flag.Int("num-nodes", -1, "custom number of nodes for the benchmark")
	customNumDKs      = flag.Int("num-dynakubes", -1, "custom number of dynakubes for the benchmark")
	customNumEntities = flag.Int("num-entities", -1, "custom number of entities for the benchmark")
)

func getCustomBenchmarkConfig(b *testing.B) benchmarkConfig {
	b.Helper()
	if *customNumNodes < 0 || *customNumDKs < 0 || *customNumEntities < 0 {
		b.Skip("custom benchmark parameters not provided")
	}

	return benchmarkConfig{
		NumNodes:     *customNumNodes,
		NumDynakubes: *customNumDKs,
		NumEntities:  *customNumEntities,
	}
}

func BenchmarkNodesController_Reconcile_custom(b *testing.B) {
	runBenchmarkReconcile(b, getCustomBenchmarkConfig(b))
}

func BenchmarkNodesController_Reconcile_small(b *testing.B) {
	runBenchmarkReconcile(b, smallBenchmarkConfig)
}

func BenchmarkNodesController_Reconcile_medium(b *testing.B) {
	runBenchmarkReconcile(b, mediumBenchmarkConfig)
}

func BenchmarkNodesController_Reconcile_large(b *testing.B) {
	runBenchmarkReconcile(b, largeBenchmarkConfig)
}

func BenchmarkNodesController_Reconcile_xlarge(b *testing.B) {
	runBenchmarkReconcile(b, xlargeBenchmarkConfig)
}

func runBenchmarkReconcile(b *testing.B, config benchmarkConfig) {
	b.Helper()
	b.Setenv("RUN_LOCAL", "true")
	b.Setenv(k8senv.PodNamespace, testNamespace)

	// 1. Setup dt server
	dtServer := config.SetupDTServerMock(b)
	defer dtServer.Close()

	// 2. Setup env
	clt := integrationtests.SetupTestEnvironment(b)

	// 3. Setup dks/nodes
	config.SetupDKs(b, clt, dtServer.URL)
	config.SetupNodes(b, clt)

	// 4. Benchmark reconcile
	// It measures the cost of reconciling nodes in the controller.
	// The i variable is used to cycle through the nodes. This is so we can utilize the b.Loop() function properly, but also ensure we do not exceed the number of nodes created.
	controller := nodes.NewControllerFromClient(clt)
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		if i >= config.NumNodes {
			i = 0
		}
		result, err := controller.Reconcile(b.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(genNode(i))})
		require.NoError(b, err)
		require.NotNil(b, result)
		i++
	}
	config.ReportMetrics(b)
}

func BenchmarkNodesController_ReconcileOnDelete_custom(b *testing.B) {
	runBenchmarkReconcileOnDelete(b, getCustomBenchmarkConfig(b))
}

func BenchmarkNodesController_ReconcileOnDelete_small(b *testing.B) {
	runBenchmarkReconcileOnDelete(b, smallBenchmarkConfig)
}

func BenchmarkNodesController_ReconcileOnDelete_medium(b *testing.B) {
	runBenchmarkReconcileOnDelete(b, mediumBenchmarkConfig)
}

func BenchmarkNodesController_ReconcileOnDelete_large(b *testing.B) {
	runBenchmarkReconcileOnDelete(b, largeBenchmarkConfig)
}

func BenchmarkNodesController_ReconcileOnDelete_xlarge(b *testing.B) {
	runBenchmarkReconcileOnDelete(b, xlargeBenchmarkConfig)
}

func runBenchmarkReconcileOnDelete(b *testing.B, config benchmarkConfig) {
	b.Helper()
	b.Setenv("RUN_LOCAL", "true")
	b.Setenv(k8senv.PodNamespace, testNamespace)

	// 1. Setup dt server
	dtServer := config.SetupDTServerMock(b)
	defer dtServer.Close()

	// 2. Setup env
	clt := integrationtests.SetupTestEnvironment(b)

	// 3. Setup dks/nodes
	config.SetupDKs(b, clt, dtServer.URL)
	config.SetupNodes(b, clt)

	// 4. Initial reconcile to ensure nodes are known to the controller
	controller := nodes.NewControllerFromClient(clt)
	for i := range config.NumNodes {
		result, err := controller.Reconcile(b.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(genNode(i))})
		require.NoError(b, err)
		require.NotNil(b, result)
	}

	// 5. Remove all the nodes, so the reconcile will process deletions
	config.RemoveNodes(b, clt)

	// 6. Benchmark reconcile when nodes are deleted
	// It measures the cost of reconciling node deletions in the controller and sending the MARK_FOR_TERMINATION event.
	// The i variable is used to cycle through the nodes. This is so we can utilize the b.Loop() function properly, but also ensure we do not exceed the number of nodes created.
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		if i >= config.NumNodes {
			i = 0
		}
		result, err := controller.Reconcile(b.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(genNode(i))})
		require.NoError(b, err)
		require.NotNil(b, result)
		i++
	}
	config.ReportMetrics(b)
}

func BenchmarkNodesController_ReconcileOnDeleteWithManager_large(b *testing.B) {
	runBenchmarkReconcileOnDeleteWithManager(b, largeBenchmarkConfig)
}

func BenchmarkNodesController_ReconcileOnDeleteWithManager_custom(b *testing.B) {
	runBenchmarkReconcileOnDeleteWithManager(b, getCustomBenchmarkConfig(b))
}

func runBenchmarkReconcileOnDeleteWithManager(b *testing.B, config benchmarkConfig) {
	b.Helper()
	b.Setenv("RUN_LOCAL", "true")
	b.Setenv(k8senv.PodNamespace, testNamespace)

	scanner := setupLogCapture(b)

	// 1. Setup dt server
	dtServer := config.SetupDTServerMock(b)
	defer dtServer.Close()

	// 2. Setup env
	clt := integrationtests.SetupManagerTestEnvironment(b, func(m controllerruntime.Manager) error { return nodes.Add(m, "") })

	// 3. Setup dks/nodes
	config.SetupDKs(b, clt, dtServer.URL)
	config.SetupNodes(b, clt)
	// 4. Initial wait for all nodes to be processed
	waitForLogMessage(b, scanner, fmt.Sprintf("node-%d", config.NumNodes-1))

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if i >= config.NumNodes {
			i = 0
		}
		require.NoError(b, clt.Delete(b.Context(), genNode(i)))
		time.Sleep(time.Second / 2)

		waitForLogMessage(b, scanner, "sending mark for termination event to dynatrace server")
		i++
	}
	config.ReportMetrics(b)
}

func setupLogCapture(b *testing.B) *bufio.Scanner {
	b.Helper()
	logs := &bytes.Buffer{}
	scanner := bufio.NewScanner(logs)
	nodes.Log = logd.CreateLogger(io.MultiWriter(logs, os.Stdout), zapcore.InfoLevel)

	return scanner
}

func waitForLogMessage(b *testing.B, scanner *bufio.Scanner, message string) {
	b.Helper()
	scannedLogs := make([]byte, 0)
	for scanner.Scan() {
		time.Sleep(time.Second / 10)
		scannedLogs = append(scannedLogs, scanner.Bytes()...)
		if strings.Contains(string(scannedLogs), message) {
			break
		}
	}
}
