# Contributing

1. Fork the dynatrace-operator repository and get the source code:

```sh
git clone https://github.com/<your_username>/dynatrace-operator
cd dynatrace-operator
```

2. Install development prerequisites:

```sh
make prerequisites
```

3. Create a new branch to work on:

```sh
git checkout -b feature/your-branch
```

4. Once the changes are finished, make sure there are no warnings on the code and [run the unit tests](#how-to-run-unit-tests) and [end-to-end tests](#how-to-run-e2e-tests) if necessary.

```sh
make go/test
make test/e2e/<your_changes_scope>
```

5. Create a pull request from the fork ([see guide](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)), with a proper title and fill out the description template. Once everything is ready, set the PR ready for review.

6. A maintainer will review the pull request and make comments. Prefer adding additional commits over amending and force-pushing since it can be difficult to follow code reviews when the commit history changes. Commits will be squashed when they're merged.

## How to run unit tests

Run the go unit tests via make:

```sh
make go/test
```

## How to run E2E tests

> **Prerequisites:**
>
> - Existing kubectl context from a test K8s cluster
> - Cleanup the cluster using `make undeploy`
> - Configured Dynatrace tenants with an access token (see `/test/testdata/secrets-samples`). Read more about Access tokens on the [official documentation](https://www.dynatrace.com/support/help/manage/access-control/access-tokens).

Check the available E2E tests via make command:

```sh
make help | grep 'e2e'
```

We recommended to only execute the ones related to the changes as each one can take some minutes to finish.

## Coding style guide

### General

- Use descriptive names (`namespace` is better than `ns`, `dynakube` is better than `dk`, etc.)
- Avoid using `client.Client` for 'getting' resources, use `client.Reader` (also known as `apiReader`) instead.
  - `client.Client` uses a cache (or tries to) that requires more permissions than normally, and can also give you outdated results.
- Do not create methods with more than two parameters (in extremely rare occasions maybe three) except constructors and factory functions. Structs and interfaces exist for a reason.
- Avoid returning responses (e.g., reconcile.Result, admission.Patched) in anything but Reconcile or Handle functions.

### Cuddling of statements

Statements must be cuddled, i.e., written as a single block, if an `if`-statement directly follows a single assignment and the condition is directly related to the assignment.
This commonly occurs with error handling, but is not restricted to it.
Example:

```go
err := assignment1()
if err != nil {
  do()
}

value1 := assignment2()
if value1 {
  do()
}
```

Statements must not be cuddled with each other if multiple of the same type of statements follow each other.
A statement must be cuddled with following statements, if they are of the same type.
Example:

```go
value1 := assignment1()
value2 := assignment2()
value3, err := assignment3()

if err != nil {
  do()
}
if value1 == "something" {
  do()
}
if value2 {
  do()
}
```

### Reconciler vs Controller

#### A **Controller** is a struct that **DIRECTLY** handles the reconcile Requests

Important characteristics:

- We pass it to `ctrl.NewControllerManagedBy(mgr)`
- Has a `Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error)` function.
- Calls other Reconcilers when needed
- Examples: DynakubeController, WebhookCertController, NodesController, OneAgentProvisioner, CSIGarbageCollector

#### A **Reconciler** is a struct that **INDIRECTLY** handles the reconcile Requests

Important characteristics:

- Has a `Reconcile(<whatever is necessary>)` function
- Is called/used BY a Controller
- Examples: OneAgentReconciler, IstioReconciler...

### Errors

#### Do's

- If an error is returned by an external function from an external package, it must be wrapped with `errors.WithStack()`.
- If an error is instantiated by internal code, it must be instantiated with `errors.New()`
- If an error is returned by an internal function and propagated, it must be propagated as is and must **not** be wrapped with `errors.WithStack()`.
  - The stacktrace is already complete when `errors.New` is called, wrapping it with `errors.WithStack()` convolutes it.

#### Don'ts

- Errors must not be logged with `log.Errorf`
  - Errors are propagated to the controller or reconciler and then automatically logged by the Operator-SDK framework

### Logging

#### Do's

- Use a package global `logger` (1 per package), should be named `log` and be declared in the `config.go` of the package. (temporary solution)
  - In case of larger packages it's advisable to introduce separate loggers for different parts of the package, these sub-loggers should still be derived from the main logger of the package and given a name.
    - Example: in webhook/mutation `var nsLog = log.WithName("namespace")` (the name of this logger is `mutation-webhook.namespace`)
- Use the logger defined in the `dynatrace-operator/src/logger` and always give it a name.
  - The name of the logger (given via `.WithName("...")`) should use `-` to divide longer names.
  - Example: `var log = logger.Factory.GetLogger("mutation-webhook")`

#### Don'ts

- Don't use `fmt.Sprintf` for creating log messages, the values you wish to replace via `Sprintf` should be provided to the logger as key-value pairs.
  - Example: `log.Info("deleted volume info", "ID", volume.VolumeID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)`
- Don't start a log message with uppercase characters, unless it's a name of an exact proper noun.

### Testing

#### Do's

- Write unit-tests ;)
- Use `assert` and `require`. (`github.com/stretchr/testify/assert, github.com/stretchr/testify/require`)
- `require` is your friend, use it for checking errors (`require.NoError(t, err)`) or anywhere where executing the rest of the `assert`s in case of the check failing would just be noise in the output.
- Abstract the setup/assert phase as much as possible so it can be reused in other tests in the package.
- Use `t.Run` and give a title that describes what you are testing in that run.
- Use this structure:

```go
func TestMyFunction(t *testing.T) {
    // Common setup, used for multiple cases
    testString := "test"

    // Each test case of a function gets a t.Run
    t.Run(`useful title`, func(t *testing.T) {
        // Arrange/Setup
        testInt := 1

        // Act
        out, err := MyFunction(testString, testInt)

        // Assert
        require.Nil(t, err)
        assert.Equal(t, out, testString)
    })

    t.Run(`other useful title`, func(t *testing.T) {
        // Arrange
        testInt := 4

        // Act
        out, err := MyFunction(testString, testInt)

        // Assert
        require.Error(t, err)
        assert.Empty(t, out)
    })
}
```

#### Don'ts

- Don't name the testing helper functions/variables/constants in a way that could be confused with actual functions. (e.g. add `test` to the beginning)
- Avoid magic ints/strings/... where possible, give names to your values and try to reuse them where possible
- Don't name test functions like `TestMyFunctionFirstCase`, instead use single `TestMyFunction` test function with multiple `t.Run` cases in it that have descriptive names.

## E2E testing guide

We are using the [e2e-framework](https://github.com/kubernetes-sigs/e2e-framework) package to write our E2E tests.

This framework allows the user a lot of ways to write/structure their tests. Therefore we had to agree on how we are going to structure our tests, otherwise it would be just a convoluted mess.

So here are some basic guidelines:

- Each `TestMain` should test a single `features.Feature`
  - Good Example: `test/scenarios/classic/classic_test.go`
  - Bad Example: `test/scenarios/cloudnative/basic/cloudnative_test.go` (should be refactored)
  - Reason: So you can easily run the tests 1-by-1.
- Test cases are defined as a single `features.Feature`
  - Reason: In a `features.Feature` you can define Setup - Assess - Teardown steps, in a nice way.
    - Having the cleanup close to the logic that creates it makes it easier to make sure that everything will be cleaned up/
    - Furthermore it makes it more understandable what a test case does.
- Don't use `Setup` step in a `features.Feature
  - Reason: If a `Setup` test fails, no other step will run, which sound fine, but this includes `Teardown` steps, which is not acceptable.
    - We run the test one after the other on the same cluster daily, so cleanup is essential.
    - So we use `Assess` steps for setting up the environment and checking it.
      - The downside of this that even if we fail during setup, still all test will run needlessly, as they will definitely fail.
      - Still better then no no cleanup.
- Use the `DynaKube` as the "single-source-of-truth" as much as possible
  - Reason: Most things that the operator deploys (name of services and pods, contents of labels, should CSI be used, etc...) depend on the `DynaKube`
  - Also the namespace of the `DynaKube` should be the same as the operator, so it can help how the operator should be deployed
  - This eliminates lots of hardcoded strings and such
- Don't reinvent the wheel, try to use what is already there.
  - If a helper function is almost fits your use case then first just try to "renovate the wheel" and make what is already there better :)

## Useful commands

### Remove all Dynatrace pods in force mode (useful debugging E2E tests)

```sh
kubectl delete pods --all --force --grace-period=0 -n dynatrace
```

### Copy CSI driver database to localhost for introspection via sqlite command

```sh
kubectl cp dynatrace/dynatrace-oneagent-csi-driver-<something>:/data/csi.db csi.sqlite
```

### Add debug suffix on E2E tests to avoid removing pods

```sh
make test/e2e/cloudnative/proxy/debug
```
