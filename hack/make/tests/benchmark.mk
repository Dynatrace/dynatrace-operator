BENCHTIME := 10x

GOBENCHCMD := go test -benchmem -benchtime=$(BENCHTIME)

NUM_NODES ?= -1
NUM_DK ?= -1
NUM_ENTITIES ?= -1
CUSTOM_NODE_ARGS := -args -num-nodes=$(NUM_NODES) -num-dynakubes=$(NUM_DK) -num-entities=$(NUM_ENTITIES)

HIDE_LOGS := | grep "BenchmarkNodesController_"

define RUN_NODE_CONTROLLER_BENCHMARK
	@$(GOBENCHCMD) -bench=$(1) -cpuprofile=$(1)_cpu.prof -memprofile=$(1)_mem.prof ./test/benchmarks/nodes_controller/... $(2) $(HIDE_LOGS)
endef

benchmark/nodes-controller/%/verbose:
	@make HIDE_LOGS="" $(@D)

benchmark/nodes-controller:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,.)

benchmark/nodes-controller/reconcile-small:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_Reconcile_small)

benchmark/nodes-controller/reconcile-medium:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_Reconcile_medium)

benchmark/nodes-controller/reconcile-large:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_Reconcile_large)

benchmark/nodes-controller/reconcile-xlarge:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_Reconcile_xlarge)

benchmark/nodes-controller/reconcile-custom:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_Reconcile_custom,$(CUSTOM_NODE_ARGS))

benchmark/nodes-controller/on-delete-small:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_ReconcileOnDelete_small)

benchmark/nodes-controller/on-delete-medium:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_ReconcileOnDelete_medium)

benchmark/nodes-controller/on-delete-large:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_ReconcileOnDelete_large)

benchmark/nodes-controller/on-delete-xlarge:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_ReconcileOnDelete_xlarge)

benchmark/nodes-controller/on-delete-custom:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_ReconcileOnDelete_custom,$(CUSTOM_NODE_ARGS))

benchmark/nodes-controller/on-delete-with-manager-custom:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_ReconcileOnDeleteWithManager_custom,$(CUSTOM_NODE_ARGS))
