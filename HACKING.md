# Dynatrace Operator

## Hack on Dynatrace Operator

[Operator SDK](https://github.com/operator-framework/operator-sdk) is the underlying framework for Dynatrace Operator. The `operator-sdk` tool needs to be installed upfront as outlined in the
[Operator SDK User Guide](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#install-the-operator-sdk-cli).

### Installation

There are automatic builds from the master branch. The latest development build can be installed as follows:

#### Kubernetes
```sh
$ make deploy
```

#### OpenShift

```sh
$ make deploy-ocp
```

#### Tests

The unit tests can be executed as follows:

```
$ go test ./...
```
