BENCHTIME := 10x

GOBENCHCMD := go test -benchmem -benchtime=$(BENCHTIME)

NUM_NODES ?= -1
NUM_DK ?= -1
NUM_ENTITIES ?= -1
CUSTOM_NODE_ARGS := -args -num-nodes=$(NUM_NODES) -num-dynakubes=$(NUM_DK) -num-entities=$(NUM_ENTITIES)

HIDE_LOGS := | grep "BenchmarkNodesController_"

benchmark/nodes-controller/%/verbose:
	@make HIDE_LOGS="" $(@D)

benchmark/nodes-controller:
	$(GOBENCHCMD) -bench=. ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/reconcile-small:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_Reconcile_small  ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/reconcile-medium:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_Reconcile_medium  ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/reconcile-large:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_Reconcile_large  ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/reconcile-xlarge:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_Reconcile_xlarge  ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/reconcile-custom:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_Reconcile_custom   ./test/benchmarks/nodes_controller/... $(CUSTOM_NODE_ARGS) $(HIDE_LOGS)

benchmark/nodes-controller/on-delete-small:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_ReconcileOnDelete_small  ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/on-delete-medium:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_ReconcileOnDelete_medium  ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/on-delete-large:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_ReconcileOnDelete_large  ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/on-delete-xlarge:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_ReconcileOnDelete_xlarge  ./test/benchmarks/nodes_controller/... $(HIDE_LOGS)

benchmark/nodes-controller/on-delete-custom:
	$(GOBENCHCMD) -bench=BenchmarkNodesController_ReconcileOnDelete_custom   ./test/benchmarks/nodes_controller/... $(CUSTOM_NODE_ARGS) $(HIDE_LOGS)
