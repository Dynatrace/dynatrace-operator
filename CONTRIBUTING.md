# Contributing

- [Steps](#steps)
- [Unit tests](#unit-tests)
- [E2E tests](#e2e-tests)
- [Useful commands](#useful-commands)
  - [Remove all Dynatrace pods in force mode (useful debugging E2E tests)](#remove-all-dynatrace-pods-in-force-mode-useful-debugging-e2e-tests)
  - [Copy CSI driver database to localhost for introspection via sqlite command](#copy-csi-driver-database-to-localhost-for-introspection-via-sqlite-command)
  - [Add debug suffix on E2E tests to avoid removing pods](#add-debug-suffix-on-e2e-tests-to-avoid-removing-pods)
  - [Debug cluster nodes by opening a shell prompt (details here)](#debug-cluster-nodes-by-opening-a-shell-prompt-details-here)

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

6. Create a pull request from the fork ([see guide](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)), with a proper title and fill out the description template. Once everything is ready, set the PR ready for review.

7. A maintainer will review the pull request and make comments. Prefer adding additional commits over amending and force-pushing since it can be difficult to follow code reviews when the commit history changes. Commits will be squashed when they're merged.

## Unit tests

Run the go unit tests via make:

```sh
make go/test
```

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

### Debug cluster nodes by opening a shell prompt ([details here](https://www.psaggu.com/upstream-contribution/2021/05/04/notes.html))

```sh
oc debug node/<node-name>
```
