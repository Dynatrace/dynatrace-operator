# How to

## Run the tests

With the `make` command
```
make kuttl
```
This will run the tests and then try to delete everything.


OR the kuttl command
```
kubectl kuttl test --config src/testing/e2e/kuttl/kuttl-test.yaml
```

## Add new tests

To add a new test case, create a folder in `./src/testing/e2e/kuttl`

To add a new test step, create 3 files in the test case folder
```
00-myStep.yaml <== Where you put the yamls to be applied
00-assert.yaml <== The expected result
00-errors.yaml <== Stuff that shouldn't happen
```
`00` stands for which step these files are for, so step 0. is `00`, step 1. is `01` etc.;
