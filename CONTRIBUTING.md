# Contributing

- [Pull requests](#pull-requests)
- [Quick start](#quick-start)
- [Unit tests](#unit-tests)
- [Integration tests](#integration-tests)
- [E2E tests](#e2e-tests)
- [Useful commands](#useful-commands)
  - [Remove all Dynatrace pods in force mode (useful debugging E2E tests)](#remove-all-dynatrace-pods-in-force-mode-useful-debugging-e2e-tests)
  - [Add debug suffix on E2E tests to avoid removing pods](#add-debug-suffix-on-e2e-tests-to-avoid-removing-pods)
  - [Debug cluster nodes by opening a shell prompt (details here)](#debug-cluster-nodes-by-opening-a-shell-prompt)

## Pull requests

Make sure all the following are true when creating a pull-request:

- The [coding style guide](doc/coding-style-guide.md) was followed when creating the change.
- The PR has a meaningful title [guidelines](https://github.com/kubernetes/community/blob/master/contributors/guide/pull-requests.md#use-imperative-mood-in-your-commit-message-subject).
- The PR is labeled accordingly with a **single** label.
- The PR has a link to a Jira ticket that requested the change. (If it exists)
- Unit tests have been updated/added.
- Relevant documentation has been updated/added.
  - [ARCHITECTURE.md](https://github.com/Dynatrace/dynatrace-operator/blob/main/ARCHITECTURE.md)
  - [Other docs](https://github.com/Dynatrace/dynatrace-operator/blob/main/doc)

### Bug fixes

There are 2 types of bugfixes, where the process differs.

But a general rule:

- Always add/update unittest when doing a bugfix, to make sure it doesn't happen again.

#### Bugfix for bugs in production

1. Must be fixed first on the (release) branch where it will be first released. On this PR use the `bug` label.
2. Cherry picked to `main` afterwards. On this PR use the `cherrypick` label.

##### Reasoning

We have the labels, but using them inconsistently makes them useless.

#### Bugfix for bugs in main (ie.: not released), example: for new features

If the change broke the e2e test run or is a "recent" change:

1. The initial PR introducing the (buggy) feature must be **reverted**.
2. (re)Implement feature with the fix in it. The bugfix must be in separate commits in the PR, at the end, these commits should have the bugfix ticket in their description.

> "recent" ~ Happened within the same week/sprint or just use common sense. 😅

If the bug flew under the radar, and got discovered later:

1. Create a PR with the fix
2. In the description of the PR, mention the original change that introduced the bug.
3. Link the related Jira bug ticket to the original implementation Jira ticket.

##### Reasoning

- We must aim to have `main` to be in a correct state by end of day. So that we do not (re)test known to be buggy state.
- Using `revert` quickly can quickly unblock others and allow the proper fix to be not rushed due to time pressure.
- Cherry-picking 1 commit is easier than cherry-picking 2 commits where the 2nd commit is the bugfix for the 1st commit, but you have several unrelated commits between them.
- Linking related changes is always nicer for tracking, and helps with cherry-picks for cases were using `revert` would have been an overkill.

## Quick start

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

    > Group your branch into a category using a prefix for your branch name, like `feature/`, `ci/`, `bugfix/`, `doc/`.

   ```sh
   git checkout -b feature/your-branch
   ```

5. Once the changes are finished, make sure there are no warnings in the code. For debugging you can [run the unit tests](#unit-tests) and [end-to-end tests](#e2e-tests).

    > **NOTE:**
    > Unit tests can also be automatically run via pre-commit hook, installed by running `make prerequisites/setup-pre-commit`.
    > With the pre-commit hook can only commit code that passes all checks.

    ```sh
    make go/test
    make test/e2e/<scope_of_the_changes>
    ```

6. To test your changes on a cluster use

    > Pushing to the default container registry (`quay.io/dynatrace/dynatrace-operator`) requires specific permissions.
    > You can use your own container registry by setting the `IMAGE` environment variable to a different value.

    1. Connect to a cluster using `kubectl`
    2. Use make commands to build and deploy your operator as follows:

    ```sh
    make build && make deploy
    ```

7. Create a pull request from the fork ([see guide](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)), with a proper title and fill out the description template. Once everything is ready, set the PR ready for review.

8. A maintainer will review the pull request and make comments. It's preferable to add additional commits over amending and force-pushing since it can be difficult to follow code reviews when the commit history changes. Commits will be squashed when they're merged.

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

Mockery is installed by running (see [docs](https://vektra.github.io/mockery/latest/installation/#go-install) for further information)

```shell
make prerequisites/mockery
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
make go/gen_mocks
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

## Integration tests

Based on [controller-runtime/pkg/envtest](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest#pkg-overview)

### setup

```bash
make integrationtest
```

### motivation

Mocking everything during unit-tests is not a good idea if we want to test some limitations of api-server
(especially different versions): e.g. But from another side e2e tests requires lots of setup
(even kind, you need to set up a cluster, deploy operator and wait when it's ready, and
only after you can try to run you test).

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

We recommended only executing the ones related to the changes as each one can take some minutes to finish.

### Triggering E2E tests on kind in CI

You can trigger the E2E tests on a kind cluster in GitHub Actions by commenting on a pull request with:

```sh
/run-e2e-kind
```

This is an optional check that won't block PR merging. It's useful for validating changes in a clean environment before merging.

## Useful commands

### Install kind cluster

```sh
K8S_VERSION=1.31 make kind/setup

# or 1.34 (default if K8S_VERSION not set)

make kind/setup
```

> [!NOTE]
> The kind cluster will be created with the name `kind`.
> and the kubeconfig context will be set to `kind-kind` (see `kind get clusters` and `kubectl config get-contexts`).
> All you need to do is to run e2e tests via `make test/e2e/<scope_of_the_changes>` against kind cluster.

### Delete kind cluster

```sh
kind delete cluster --name kind
```

### Remove all Dynatrace pods in force mode (useful debugging E2E tests)

```sh
kubectl delete pods --all --force --grace-period=0 -n dynatrace
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
