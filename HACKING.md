# Dynatrace Operator

## Installation

There are automatic builds from the main branch. The latest development build can be installed using `make deploy`.

### Tests

The unit tests can be executed as follows:

```sh
make go/test
```

### Run Benchmarks

#### Nodes controller benchmarks

You can run the benchmarks for the nodes controller using the following commands:

```sh
make benchmark/nodes-controller
```

- This will execute all benchmarks related to the nodes controller.

```sh
make benchmark/nodes-controller/reconcile
```

- This will specifically run the benchmarks for the reconcile scenario of the nodes controller.
  - We only run the `Reconcile` func against an env were no node deletion happened.

```sh
make benchmark/nodes-controller/on-delete
```

- This will specifically run the benchmarks for the on-delete scenario of the nodes controller.
  - We only run the `Reconcile` func against an env were node deletions have occurred -> So each reconcile will process node deletions.

##### Results

The benchmark results will be displayed in the terminal after the execution:

```sh
$ make benchmark/nodes-controller/reconcile
BenchmarkNodesController_Reconcile-12                 10          91169608 ns/op                 1.000 dynakubes                10.00 host-entities             10.00 nodes       107100 B/op        960 allocs/op
```

`.prof` files will also be generated, that contain the profiling data for CPU and memory usage. You can analyze these files using the `go tool pprof` command.

- The files will be named after the scenario with the values used in the benchmark and will be located in the current (ie.: project root) directory.

```sh
BenchmarkNodesController_Reconcile_10n_1d_10e_cpu.prof
BenchmarkNodesController_Reconcile_10n_1d_10e_mem.prof

# 10n == 10 nodes, 1d == 1 dynakube, 10e == entities
```

- Disclaimer: The `.prof` profiling data does not consider the `b.Loop()` iterations separately, so the data will be cumulative for all iterations + setup.

##### Customize Runs

You can customize the benchmark runs by calling them the following way:

```sh
NUM_NODES=1000 NUM_DK=1 NUM_ENTITIES=1000 make benchmark/nodes-controller/on-delete
```

- `NUM_NODES`: Number of nodes to simulate in the benchmark (default: 10)
  - Do not have less nodes than `BENCHTIME` (default: 10x) to avoid skewed results.
    - One node will be processed during one `Reconcile` loop, so if you have less nodes than loops, some nodes will be processed multiple times, which can skew the results.
- `NUM_DK`: Number of Dynakube instances to simulate (default: 1)
- `NUM_ENTITIES`: Number of host entities to simulate on the mocked Dynatrace API (default: 10)

##### Show/hide logs

By default you only see the benchmark results. If you want to see the logs during the benchmark execution, use the `/verbose` suffix:

```sh
make benchmark/nodes-controller/on-delete/verbose
```

##### Disclaimer

- These benchmarks are not using the `Manager` from controller-runtime, but are instantiating the controller directly. This means that some of the setup and background tasks that would normally be handled by the `Manager` are not present in these benchmarks. The focus is on measuring the performance of the controller's logic itself.
  - I did some POC tests with a `Manager` as well, but no significant difference was observed in the results, however the logic was more complex to setup and control.
