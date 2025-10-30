# Coding style guide

- [General](#general)
- [Function Parameter and Return-Value Order](#function-parameter-and-return-value-order)
- [Cuddling of statements](#cuddling-of-statements)
- [Reconciler vs Controller](#reconciler-vs-controller)
  - [A **Controller** is a struct that **DIRECTLY** handles the reconcile Requests](#a-controller-is-a-struct-that-directly-handles-the-reconcile-requests)
  - [A **Reconciler** is a struct that **INDIRECTLY** handles the reconcile Requests](#a-reconciler-is-a-struct-that-indirectly-handles-the-reconcile-requests)
- [Errors](#errors)
  - [Do's](#dos)
  - [Don'ts](#donts)
- [Naming](#naming)
  - [Do's](#dos-1)
  - [Don'ts](#donts-1)
- [Logging](#logging)
  - [Do's](#dos-1)
  - [Don'ts](#donts-1)
  - [Debug logs](#debug-logs)
- [Testing](#testing)
  - [Do's](#dos-2)
  - [Don'ts](#donts-2)
- [E2E testing guide](#e2e-testing-guide)
- [Code Review](#code-review)

## General

- Use descriptive (variable) names
  - Shortnames for known Kubernetes Objects are fine. (`ns` for namespace)
  - Avoid "stuttering". (In the `beepboop` package don't call you `struct` `BeepBoopController`, but just `Controller`)
    - Relevant for: folder/package, file, struct, func and const/variable names.
  - Do NOT shadow builtin names and packages.
- Avoid using `client.Client` for 'getting' resources, use `client.Reader` (also known as `apiReader`) instead.
  - `client.Client` uses a cache (or tries to) that requires more permissions than normally, and can also give you outdated results.
- Avoid creating functions with more than 3 params, except constructors and factory functions. Structs and interfaces exist for a reason.
- Avoid returning responses (e.g., reconcile.Result, admission.Patched) in anything but Reconcile or Handle functions.
- Run the linters locally before opening a PR, it will save you time.
  - There is a pre-commit hook that you can configure via `make prerequisites/setup-pre-commit`
- Avoid using the `path` package for operations on filesystem paths. Use "path/filepath" package.

## Function Parameter and Return-Value Order

Ordering of **function parameters** should be:

1. Convention
   - example: `ctx context.Context`
2. Interfaces
   - example: `kubeClient client.Client`
3. Data Structs
   - example: `pod corev1.Pod`
4. Simple/base types
   - example: `data string`

Ordering of **return values** is more straightforward; the `err error` should always be the last, and AVOID returning more than two values. If more than two return values are needed, try splitting the logic or collecting the return values in a `struct`.

So a full example: `func ExampleFunc(ctx context.Context, kubeClient client.Client, pod corev1.Pod, data string) (corev1.Pod, error) {...}`

## Cuddling of statements

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

## Go struct field alignment

Short summary what this is:

> Field alignment in Go refers to how the memory layout of `struct` fields is organized to meet specific alignment requirements, which can improve performance and reduce memory usage. Proper alignment ensures that fields are stored at memory addresses that are multiples of their size, minimizing padding and optimizing access speed on modern CPUs.

And why you do not need to use it:

- The performance benefits for most cases are negligible, or even undetectable.
- Ordering the fields in a `struct` for "cpu efficiency" can/will sacrifice "human understandability" of the `struct`.
  - In the majority of the cases, we value "human understandability" over "cpu efficiency".

When to use it:

- If you have a scenario (for specific `structs`), where you **can back it up with benchmarks** that it makes a significant difference.

## Reconciler vs Controller

### A **Controller** is a struct that **DIRECTLY** handles the reconcile Requests

Important characteristics:

- We pass it to `ctrl.NewControllerManagedBy(mgr)`
- Has a `Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error)` function.
- Calls other Reconcilers when needed
- Examples: DynakubeController, WebhookCertController, NodesController, OneAgentProvisioner, CSIGarbageCollector

### A **Reconciler** is a struct that **INDIRECTLY** handles the reconcile Requests

Important characteristics:

- Has a `Reconcile(<whatever is necessary>)` function
- Is called/used BY a Controller
  - Examples: OneAgentReconciler, IstioReconciler...
- `Reconciler`s don't hold state in a way that they need to be passed around, or if they currently do, they shouldn't.
  - Configuring a `struct` once a reusing it makes sense IF the creation was costly. (example: required an API call)
    - A `Reconciler` shouldn't be this kind of struct, it should be "throw away struct."
- The cleanup for the resources that are created by the `Reconciler` are cleaned up by the same `Reconciler`
  - The cleanup is "invisible" for the caller of the `Reconciler`, so its up to the `Reconciler` to decide if it needs to clean up or setup/update.
    - Example: `DynakubeController` should just call the `ActiveGateReconciler.Reconcile` function, and not worry about if it will create(i.e.: setup) a `StatefulSet` or delete(i.e.: cleanup) the no longer necessary one.

## Secret/ConfigMap handling

A significant role of the Operator is to create/update or just pass along Secrets/ConfigMaps to components it manages.

The main problem that needs to be solved is: If a `Secret/ConfigMap` changes, that warrants the restart of a component that uses it, how do we go about this?

General rules to follow:

- Each `Secret/ConfigMap` we manage should have it's own package, that contain:
  - The func to determine the name of the `Secret/ConfigMap`. (example: `GetName`)
  - A `const` for each key used inside the `data` of the `Secret/ConfigMap`.
    - Make them `public` if they will be used via `Items` when mounting it.
  - Unique annotation keys ( public `const`) for keeping track of `Secret/ConfigMap` changes (example: `TokenHashAnnotationKey`)
    - An annotation can be for storing a hash of the whole `Secret/ConfigMap` or part of it. Make this clear in the name of the annotation.
      - For not confidential values it is ok to use `GenerateHash`
      - For confidential values `GenerateSecureHash` should be used
    - An annotation can be for storing the exact value from the `DynaKube`, makes sense for simple, non-confidential data. (example: `networkZone`)
  - A func to add all the relevant annotations to the provided map. (example: `AddAnnotations`)
    - There can be multiple annotation for `Secret/ConfigMap`, or just 1.
      - It makes sense to have multiple for the cases where you don't want to cause a restart if part of the `Secret/ConfigMap` changes.
         Example: The communication endpoints of the OneAgent changes overtime, the OneAgent handles it on its own so that shouldn't cause a restart.
      - It makes sense to have 1 if any change to the `Secret/ConfigMap` should cause a restart.
        - Be mindful, if something is later added to such `Secret/ConfigMap`, having a single hash for the whole will cause everything using it to restart after an Operator update.
      - It is recommended to have 1 hash for all the "required" parts of the `Secret/ConfigMap`, and individual ones for the "optional" parts.
        - The "optional" parts will always be a value directly from the `DynaKube` or something referenced from the `DynaKube`, in either case there should already be a raw value or hash ready to be used.
  - The "reconcile" logic
    - If a unique hash will be calculated for this Secret, then the "reconcile logic" should calc/set/unset this as it is necessary.
      - Should be set/stored in the `Status` of the `DynaKube` so it can be reused, instead of recalculated from other reconcilers.

- This package is used by other packages that would like to mount the `Secret/ConfigMap`
  - Using the `GetName`, to reference the `Secret/ConfigMap` in a Volume or Env
  - Using the "public `const` for each key" to what to part of the `Secret/ConfigMap` to use. (use `Items` instead of `SubPath`)
  - Using the `AddAnnotation` to put the hashed content of the `Secret/ConfigMap` (from the `DynaKube`'s `status`) into the `template.metadata.annotations` of the component, **IF** it needs to be restarted when the `Secret/ConfigMap` changes.

Reasoning for these rules:

- As of writing this, we have 15+ `Secrets/ConfigMaps` that we manage and provide to 4+ components. (some components need more, some less)
  - Putting all of this within the `api` package, when we already tend to have a package for each `Secret/ConfigMap` for "reconciling" is just breaking the "single-place-of-truth" for no good reason.
  - Fetching and recalculating the hash for each component where a `Secret/ConfigMap` is used would be fine if we only had a few of them, but we have 15+.
    - Why not use the cache? -> Using the built-in cache of the `Client` had caused problems in the past (the cache was not up to date), which could lead to inconsistency.

### Example

Do this:

```go
package configsecret

// import (...)

const (
  logMonitoringSecretSuffix = "-logmonitoring-config"

  TokenHashAnnotationKey   = api.InternalFlagPrefix + "tenant-token-hash"
  NetworkZoneAnnotationKey = api.InternalFlagPrefix + "network-zone"

  //...
)

// type Reconciler struct { ... }
// func NewReconciler(...) {...}
// func (r *Reconciler) Reconcile(ctx context.Context) error {...}

func GetSecretName(dkName string) string {
  return dkName + logMonitoringSecretSuffix
}

func AddAnnotations(source map[string]string, dk dynakube.DynaKube) map[string]string {
  annotation := map[string]string{}
  if source != nil {
   annotation = maps.Clone(source)
  }

  annotation[TokenHashAnnotationKey] = dk.OneAgent().ConnectionInfoStatus.TenantTokenHash
  if dk.Spec.NetworkZone != "" {
   annotation[NetworkZoneAnnotationKey] = dk.Spec.NetworkZone
  }

  return annotation
}

```

Not this:

```go
package oneagent

// import (...)

const (
  OneAgentTenantSecretSuffix            = "-oneagent-tenant-secret"
  OneAgentConnectionInfoConfigMapSuffix = "-oneagent-connection-info"
// ...
)

func (oa *OneAgent) GetTenantSecret() string {
  return oa.name + OneAgentTenantSecretSuffix
}

func (oa *OneAgent) GetConnectionInfoConfigMapName() string {
  return oa.name + OneAgentConnectionInfoConfigMapSuffix
}

// ...
```

## Errors

### Do's

- Follow on how to [working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- Expected errors should be designed as error values (sentinel errors): `var ErrFoo = errors. New ("foo")`.

>[Sentinel errors](https://stackoverflow.com/questions/73433300/what-is-the-difference-between-errors-and-sentinel-errors) are user defined errors that indicated very specific events that you, as a developer, anticipate & identify as adequately important to define and specify.
As such, you declare them at the package level and, in doing so, imply that your package functions may return these errors (thereby committing you in the future to maintain these errors as others depending on your package will be checking for them).

- Unexpected errors should be designed as error types: `type BarError struct { ...}`, with `BarError` implementing the error interface.
- If an error is returned by an external function from an external package, it must be wrapped with `errors.WithStack()`.
- If an error is instantiated by internal code, it must be instantiated with `errors.New()`
- If an error is returned by an internal function and propagated, it must be propagated as is and must **not** be wrapped with `errors.WithStack()`.
  - The stacktrace is already complete when `errors.New` is called, wrapping it with `errors.WithStack()` convolutes it.
- If the error is not propagated to the controller or reconciler, it should be logged at the point where it is not returned to the caller.

### Don'ts

- Errors that are propagated to the controller or reconciler must not be logged directly by us, as they get automatically logged by the Operator-SDK framework.
  - So we do not log errors twice.
  - Example:

```go
// Doing something like this:
err = errors.New("BOOM!")
if err != nil {
    log.Error(err, "it happened")
    return reconcile.Result{}, err
}
// Will result in: (shortened it a bit so its not huge)
{"level":"info","ts":"2023-11-20T09:25:16.261Z","logd":"automatic-api-monitoring","msg":"kubernetes cluster setting already exists","clusterLabel":"dynakube","cluster":"a9ef1d81-6950-4260-a3d4-8e969c590b8c"}
{"level":"info","ts":"2023-11-20T09:25:16.273Z","logd":"dynakube","msg":"activegate statefulset is still deploying","dynakube":"dynakube"}
{"error":"BOOM!","level":"error","logd":"dynakube","msg":"it happened","stacktrace":"BOOM!
github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube.(*Controller).reconcile
github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynakube_controller.go:168
   <...>
sigs.k8s.io/controller-runtime@v0.16.3/pkg/internal/controller/controller.go:227
runtime.goexit
runtime/asm_amd64.s:1650","ts":"2023-11-20T09:25:16.273Z"}
{"DynaKube":{"name":"dynakube","namespace":"dynatrace"},"controller":"dynakube","controllerGroup":"dynatrace.com","controllerKind":"DynaKube","error":"BOOM!","level":"error","logd":"main","msg":"Reconciler error","name":"dynakube","namespace":"dynatrace","reconcileID":"5d67fe9e-b6f0-4ad4-8634-aa66838aa685","stacktrace":"BOOM!
github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube.(*Controller).reconcile
github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynakube_controller.go:168
   <...>
sigs.k8s.io/controller-runtime@v0.16.3/pkg/internal/controller/controller.go:227
runtime.goexit
```

## Naming

### Do's

- Use `dynakube` or `edgeconnect` as default package name in imports to simplify CRD version maintenance.
For example:

```go
package abc

import (
    "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)
```

```go
package abc

import (
    "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)
```

> Note: Incase multiple versions of the same CR are used (conversion, webhook manager configuration, etc.),
> use the name of the CR as part of the package alias:

```go
package abc

import (
    dynakubelatest  "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
    dynakubev1beta5  "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
)
```

```go
package abc

import (
    edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
    edgeconnectv1alpha2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)
```

- Use `dk` or `ec` as short a version of variable name for instances of `dynakube.Dynakube`, or any func arguments
to not overlap with package name `dynakube` or `edgeconnect`.

For example:

```go
dk := dynakube.DynaKube{}
```

```go
ec := edgeconnect.EdgeConnect{}
```

```go
func (c component) getImage(dk *dynakube.DynaKube) (string, bool) {}
```

### Don'ts

- Use CRD version inside import alias name:

```go
package abc

import (
    dynatracelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)
```

- Use `dynakube` or `edgeconnect` as variable name or function argument name.
- Use import alias for api package: `dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"`

## Logging

### Do's

- Use a package global `logger` (1 per package), should be named `log` and be declared in the `config.go` of the package. (temporary solution)
  - In case of larger packages it's advisable to introduce separate loggers for different parts of the package, these sub-loggers should still be derived from the main logger of the package and given a name.
    - Example: in webhook/mutation `var nsLog = log.WithName("namespace")` (the name of this logger is `mutation-webhook.namespace`)
- Use the logger defined in the `dynatrace-operator/src/logger` and always give it a name.
  - The name of the logger (given via `.WithName("...")`) should use `-` to divide longer names.
  - Example: `var log = logger.Get().WithName("mutation-webhook")`

### Don'ts

- Don't use `fmt.Sprintf` for creating log messages, the values you wish to replace via `Sprintf` should be provided to the logger as key-value pairs.
  - Example: `log.Info("deleted volume info", "ID", volume.VolumeID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)`
- Don't start a log message with uppercase characters, unless it's a name of an exact proper noun.

### Debug logs

- Do not log errors that bubble up to the controller runtime. They will be logged anyway, and we do not want to log errors multiple times because it's confusing.
- Use debug logs to show the flow.
- Provide metadata so that logs can be related to objects under reconciliation
- Provide additional information that might be helpful for troubleshoot in key/values of log (e.g. values of variables at that point)
- Use a local logger with pre-configured key/values to avoid duplication
- Be careful not to log confidential info like passwords or tokens accidentally.

#### Show the flow

```go
log.Debug("reconcile required", "updater", updater.Name())
```

#### Something's wrong

```go
if err != nil {
    log.Debug("could not create or update deployment for EdgeConnect", "name", desiredDeployment.Name)

    return err
}

log.Debug("EdgeConnect deployment created/updated", "name", edgeConnect.Name)
```

#### Logging additional info

```go
if len(ecs.EdgeConnects) > 1 {
    log.Debug("Found multiple EdgeConnect objects with the same name", "count", ecs.EdgeConnects)

    return edgeconnect.GetResponse{}, errors.New("many EdgeConnects have the same name")
}

```

#### Pre-configured local logger

```go
func (controller *Controller) reconcileEdgeConnectDeletion(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
    _log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

    ...
    _log.Debug("foobar happened")

    _log = _log.WithValues("foobarReason", "hurga")

    ...
    _log.Debug("foobar happened again")
    ...
}

```

## Testing

### Do's

- Write unit-tests ;)
- Name test functions like the following
  - for public funcs -> `func TestPublicFunc(t *testing.T)`
  - for private funcs -> `func Test_privateFunc(t *testing.T)`
  - for public method for public struct -> `func Test_MyStruct_PublicMethod(t *testing.T)`
  - for private method for public struct -> `func Test_MyStruct_privateMethod(t *testing.T)`
  - for public method for private struct -> `func Test_myStruct_PublicMethod(t *testing.T)`
  - for private method for private struct -> `func Test_myStruct_privateMethod(t *testing.T)`
  - Avoid "scenarios" in the name, use `t.Run` within the tests.
    - So avoid funcs like `func Test_MyStruct_PublicMethod_Scenario1(t *testing.T)` and instead use `t.Run("scenario1", ...)` within `func Test_MyStruct_PublicMethod`
- Use `assert` and `require`. (`github.com/stretchr/testify/assert, github.com/stretchr/testify/require`)
- `require` is your friend, use it for checking errors (`require.NoError(t, err)`) or anywhere where executing the rest of the `assert`s in case of the check failing would just be noise in the output.
- Mock per test, not per package.
  - Having one `createTestReconciler(p1,p2,p3,p4,p5)` that tries to mock all and fit in all test cases is just maintenance hell, because if you change 1 test-case then you will change all others, also your mocks will be too forgiving, which is great for bugs.
    - Create small, test specific mocks.
    - Such mocks can be created with helper functions defined within the test case, to reuse it within the test case.
- Helper functions per-test are preferred to helper functions per-package
  - Writing helper functions for a whole package can backfire quickly as it binds tests together unnecessary.
    - example: Change 1 helper function -> break 3 tests but fix 2 -> probably should be split into test specific helpers
    - For simpler common setups func it can still make sense.
  - All helper func must use `t.Helper()` as their first-line, for improved error reporting.
- Defining (testing) `consts` per-test is preferred to defining (testing) `consts` per-package
  - Figuring out what is `myTestConst52` that is defined 3 files over and used by 12 tests is not fun to maintain and easy to follow.
- Use `t.Run` and give a title that describes what you are testing in that run.
- Use `t.Context` instead of `context.Background`.
  - [Added in go 1.24](https://pkg.go.dev/testing#T.Context), so we don't really use it, update it where you change code
- Use `<...>mock` as package import alias, in all cases, even if no alias would strictly be necessary.
  - Examples: `dtclientmock`, `controllermock`, `dtbuildermock`, `injectionmock`, `registrymock`
- Use this structure: (or table-tests)
  - The usage of `"` instead of ``` ` ``` in `t.Run` is important, as in VSCode you can't run individual tests if they are defined as ``` t.Run(`test`, ...) ```, but can when defined as``` t.Run("test", ...) ```.

```go
func TestMyFunction(t *testing.T) {
    // Common setup, used for multiple cases
    const testString = "test"

    mySetupFunc := func(t *testing.T, prop int) {
      t.Helper()
      // Do some test specific setup/helper logic
      // - can be reused per `t.Run`
      // - can be just configurable enough to meet the requirements of the `t.Run`s in this test
      // - "hidden" from other tests, so reusing it is more intentional and wont affect this test
    }

    mockMyClient := func(t *testing.T, prop int) MyClient {
      t.Helper()
      // Creating MyClient mock, specific to this test.
      // - can be reused per `t.Run`
      // - can be just configurable enough to meet the requirements of the `t.Run`s in this test
      // - "hidden" from other tests, so reusing it is more intentional and wont affect this test
    }

    // Each test case of a function gets a t.Run
    t.Run("useful title", func(t *testing.T) {
        // Arrange/Setup
        testInt := 1
        mySetupFunc(testInt)
        client := mockMyClient(testInt)

        // Act
        out, err := MyFunction(t.Context(), client, testString, testInt)

        // Assert
        require.Nil(t, err)
        assert.Equal(t, out, testString)
    })

    t.Run("another useful title", func(t *testing.T) {
        //...
    })
}
```

### Don'ts

- Don't name the testing helper functions/variables/constants in a way that could be confused with actual functions. (e.g. add `test` to the beginning)
- Avoid magic ints/strings/... where possible, give names to your values and try to reuse them where possible
- Don't use `.Maybe()` when mocking, have test specific mocks, so the calls are checked for you.

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

## Code Review

[Common guidelines](https://go.dev/wiki/CodeReviewComments)

- (🧑‍🤝‍🧑) 2 approvals per PR is preferred
- (✅) Resolving a comment is the duty of the commenter. (after the comment was addressed)
- (😬) When nitpicking/complaining always provide possible solutions, otherwise avoid commenting about it.
- (🧑‍💻) Run the PR locally/on-your-environment if possible.
  - (🚨) If testing steps not-clear/not-provided notify the creator to improve them
  - (🚦) After running it "locally" notify the creator by commenting `Ran it, practically LGTM` (or something similar) or `Found possible issue ...`
- (📝) Enforce the coding-style-guide, by linking to it. (to the specific line)
  - (🙋) If you feel something is missing/wrong in the style-guide, discuss it with the team, and create PR for it if it was accepted.
- (💥) The `Update branch` button should be pressed only by the creator of the PR, so the reviewer does not cause unexpected `Push rejected` for the creator.
