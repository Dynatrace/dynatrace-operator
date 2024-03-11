# Contributing

- [Steps](#steps)
- [Unit tests](#unit-tests)
- [E2E tests](#e2e-tests)
- [Useful commands](#useful-commands)
  - [Remove all Dynatrace pods in force mode (useful debugging E2E tests)](#remove-all-dynatrace-pods-in-force-mode-useful-debugging-e2e-tests)
  - [Copy CSI driver database to localhost for introspection via sqlite command](#copy-csi-driver-database-to-localhost-for-introspection-via-sqlite-command)
  - [Add debug suffix on E2E tests to avoid removing pods](#add-debug-suffix-on-e2e-tests-to-avoid-removing-pods)
  - [Debug cluster nodes by opening a shell prompt (details here)](#debug-cluster-nodes-by-opening-a-shell-prompt)

## Steps

1. Read the [coding style guide](doc/coding-style-guide.md).

2. Fork the dynatrace-operator repository and get the source code:

    ```sh
    git clone https://github.com/<your_username>/dynatrace-operator
    cd dynatrace-operator
    ```

3. Install development prerequisites:

   ```sh
   make prerequisites
   ```

4. Create a new branch to work on:

   ```sh
   git checkout -b feature/your-branch
   ```

5. Once the changes are finished, make sure there are no warnings on the code. For debugging you can [run the unit tests](#unit-tests) and [end-to-end tests](#e2e-tests).

    > **NOTE:**
    > Unit tests are always executed via pre-commit hook (installed on previous steps). Meaning, you can only commit code that passes all unit tests.

    ```sh
    make go/test
    make test/e2e/<scope_of_the_changes>
    ```

6. To test your changes on a cluster use

    1. kubectl to connect to a cluster
    2. Use make commands to build and deploy your operator as follows:

    ```sh
    make build && make deploy
    ```

    >**NOTE:**
    > When building on ARM machines (such as Apple M1) podman/docker uses the local architecture if not specified otherwise.
    > To override set the ENV var OPERATOR_DEV_BUILD_PLATFORM to the desired platform (e.g. linux/amd64).
    >
    > ```shell
    >    export OPERATOR_DEV_BUILD_PLATFORM=linux/amd64
    > ```

7. Create a pull request from the fork ([see guide](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)), with a proper title and fill out the description template. Once everything is ready, set the PR ready for review.

8. A maintainer will review the pull request and make comments. Prefer adding additional commits over amending and force-pushing since it can be difficult to follow code reviews when the commit history changes. Commits will be squashed when they're merged.

## Unit tests

Run the go unit tests via make:

```sh
make go/test
```

### Mocking

For our mocking needs we trust in [testify](https://github.com/stretchr/testify) while using [mockery](https://github.com/vektra/mockery) to generate our mocks.
We check in our mocks to improve code readability especially when reading via GitHub and to remove the dependency on make scripts to run our tests.
Mockery only has to be run when adding new mocks or have to be updated to changed interfaces.

#### Installing _mockery_

Mockery is installed by either running (see [docs](https://vektra.github.io/mockery/latest/installation/#go-install) for further information)

```shell
go install github.com/vektra/mockery/v2@v2.33.2
```

or simply calling our make target:

```shell
make prerequisites
```

#### Adding a mock

When adding a mock you have to add the mocked interface to .mockery.yaml.
Take the following example of the builder package with the interfaces `Builder` and `Modifier`:

```yaml
quiet: False
disable-version-string: True
with-expecter: True
mockname: "{{.InterfaceName}}"
filename: "{{.MockName}}.go"
outpkg: mocks
dir: "test/mocks{{.InterfaceDirRelative}}"
packages:
  github.com/Dynatrace/dynatrace-operator/pkg/util/builder:
    config:
      recursive: true
     # all: true // or use all if mocks for all interfaces in a package/dir should be created
    interfaces:
      Builder:
      Modifier:
```

then run mockery by simple running

```shell
mockery
```

or

```shell
make go/gen_mock
```

#### Migrating to Mockery

To move our existing codebase to mockery you have to look out for these pitfalls:

1. As a rule o thumb, use `mocks.NewXYZ(t)` function instead of `mocks.XYZ{}` struct when any expectation is defined (`On(..)`). It allows to easily detect cases when no expectations are need or new ones should be added.

2. Mocks require a reference parameter to `testingT`:

   ```go
   //...
   b := GenericBuilder[mocks.Data]{}

   modifierMock := mocks.NewModifier[mocks.Data](t) // <- t required here
   //...
   ```

3. Add call to `Maybe()` to return if it should be tested if the function is called at all:

    ```go
    modifierMock.On("Modify", mock.Anything).Return(nil).Maybe()
    modifierMock.On("Enabled").Return(false)

    actual, _ := b.AddModifier(modifierMock).Build()
    modifierMock.AssertNotCalled(t, "Modify")
    //modifierMock.AssertNumberOfCalls(t, "Modify", 0)
   ```

  > ❗ In the case of using multiple mock packages in the same test file, the standard package alias naming is `{struct}mock`, e.g. `clientmock`.
  >
  > ```go
  > clientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
  > installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
  > reconcilermock "github.com/Dynatrace/dynatrace-operator/test/mocks/sigs.k8s.io/controller-runtime/pkg/reconcile"
  > ```

## E2E tests

> **Prerequisites:**
>
> - Existing kube config with the context of a test K8s cluster
> - Cleanup the cluster using `make undeploy`
> - Configured Dynatrace tenant(s) with an access token (see `/test/testdata/secrets-samples`). Read more about Access tokens on the [official documentation](https://www.dynatrace.com/support/help/manage/access-control/access-tokens).

Check the available E2E tests via make command:

```sh
make help | grep 'e2e'
```

We recommended to only execute the ones related to the changes as each one can take some minutes to finish.

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

### Debug cluster nodes by opening a shell prompt

[Details here](https://www.psaggu.com/upstream-contribution/2021/05/04/notes.html)

```sh
oc debug node/<node-name>
```
