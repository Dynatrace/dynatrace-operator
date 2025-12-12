package nodes_test

import (
	"flag"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/nodes"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	customNumNodes    = flag.Int("num-nodes", 10, "custom number of nodes for the benchmark")
	customNumDKs      = flag.Int("num-dynakubes", 1, "custom number of dynakubes for the benchmark")
	customNumEntities = flag.Int("num-entities", 10, "custom number of entities for the benchmark")
)

func getBenchmarkConfig(b *testing.B) benchmarkConfig {
	b.Helper()

	return benchmarkConfig{
		NumNodes:     *customNumNodes,
		NumDynakubes: *customNumDKs,
		NumEntities:  *customNumEntities,
	}
}

func BenchmarkNodesController_Reconcile(b *testing.B) {
	runBenchmarkReconcile(b, getBenchmarkConfig(b))
}

func runBenchmarkReconcile(b *testing.B, config benchmarkConfig) {
	b.Helper()
	b.Setenv("RUN_LOCAL", "true")
	b.Setenv(k8senv.PodNamespace, testNamespace)

	// 1. Setup dt server
	dtServer := config.SetupDTServerMock(b)
	defer dtServer.Close()

	// 2. Setup env
	clt := integrationtests.SetupTestEnvironment(b, integrationtests.DisableAttachControlPlaneOutput())

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

func BenchmarkNodesController_OnDelete(b *testing.B) {
	runBenchmarkOnDelete(b, getBenchmarkConfig(b))
}

func runBenchmarkOnDelete(b *testing.B, config benchmarkConfig) {
	b.Helper()
	b.Setenv("RUN_LOCAL", "true")
	b.Setenv(k8senv.PodNamespace, testNamespace)

	// 1. Setup dt server
	dtServer := config.SetupDTServerMock(b)
	defer dtServer.Close()

	// 2. Setup env
	clt := integrationtests.SetupTestEnvironment(b, integrationtests.DisableAttachControlPlaneOutput())

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
