# How to

## General Info

### `TestSuites`
A folder that contains other folders, each of those is a `TestCase`.
- To configure a `TestSuite` you have to create/update the yaml with a config. (example: `oneagent/oneagent-test.yaml`)
- `TestSuites` can be run individually, unlike the `TestCases`.

### `TestCases`
A folder that contains TestSteps each of which is a combination of 3(+) yaml files.

- `TestCases` can be run in parallel (configurable in the TestSuite)
- `TestCases` are not run in order, the order is semi-random

### `TestSteps`
A collection of yaml fines within a `TestCase`.

```
00-myStep.yaml <== Where you put the yamls to be applied (you can split this up into multiple files)
00-assert.yaml <== The expected result (this MUST exist)
00-errors.yaml <== Stuff that shouldn't happen (this can be omitted)
```

`00` stands for which step these files are for, so step 0. is `00`, step 1. is `01` etc.;

### create-dynakube-base.sh
Due to the fact you can only use envvars is shell scripts

but we need to set the api-url and tokens for the test to run (without hard-coding it in),

so this script was created to do just that.

## OneAgent Modes TestSuite
The config(`TestSuite`) for the OneAgent kuttl tests are in `oneagent/oneagent-test.yaml`.

It tests:
- The install (is everything there that we need)
- Each mode (app, cloud, classic, host-monitoring)

If you want to use `Kind` you can enable it there. (It will use the `kind.yaml` to create the local `Kind` cluster)

### Run

With the `make` command (uses `make deploy` to deploy the yamls ==> doesn't work with `startKind`)
```
make kuttl-oneagent
```
This will run the tests and then try to delete everything.


OR the kuttl command (you HAVE TO deploy the operator before by hand)
```
kubectl kuttl test --config src/testing/kuttl/oneagent/oneagent-test.yaml
```

