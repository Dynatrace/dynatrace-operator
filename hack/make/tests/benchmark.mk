BENCHTIME := 10x

GOBENCHCMD := go test -benchmem -benchtime=$(BENCHTIME)

NUM_NODES ?= 10
NUM_DK ?= 1
NUM_ENTITIES ?= 10
NODE_BENCHMARK_CONFIG := -args -num-nodes=$(NUM_NODES) -num-dynakubes=$(NUM_DK) -num-entities=$(NUM_ENTITIES)

HIDE_LOGS := | grep "BenchmarkNodesController_"

define RUN_NODE_CONTROLLER_BENCHMARK
	@$(GOBENCHCMD) -bench=$(1) -cpuprofile=$(1)_cpu.prof -memprofile=$(1)_mem.prof ./test/benchmarks/nodes_controller/... $(NODE_BENCHMARK_CONFIG) $(HIDE_LOGS)
endef

benchmark/nodes-controller/%/verbose:
	@make HIDE_LOGS="" $(@D)

benchmark/nodes-controller: benchmark/nodes-controller/reconcile benchmark/nodes-controller/on-delete

benchmark/nodes-controller/reconcile:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_Reconcile)

benchmark/nodes-controller/on-delete:
	$(call RUN_NODE_CONTROLLER_BENCHMARK,BenchmarkNodesController_ReconcileOnDelete)
