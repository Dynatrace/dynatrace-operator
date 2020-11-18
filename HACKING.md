# Dynatrace Operator

## Hack on Dynatrace Operator

[Operator SDK](https://github.com/operator-framework/operator-sdk) is the underlying framework for Dynatrace Operator. The `operator-sdk` tool needs to be installed upfront as outlined in the
[Operator SDK User Guide](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#install-the-operator-sdk-cli).

### Installation

There are automatic builds from the master branch. The latest development build can be installed on Kubernetes with,

#### Kubernetes
```sh
$ kubectl create namespace dynatrace
$ kubectl apply -k github.com/Dynatrace/dynatrace-operator/deploy/manifest
```

#### OpenShift
```sh
$ oc adm new-project --node-selector="" dynatrace
$ oc apply -k github.com/Dynatrace/dynatrace-operator/deploy/manifest
```

#### Tests

The unit tests can be executed with

```
$ go test ./...
```

And integration tests,

```
$ go test -tags integration ./...
```

These integration tests also require Kubebuilder, unpack the binaries from [the release package](https://github.com/kubernetes-sigs/kubebuilder/releases/download/v1.0.8/kubebuilder_1.0.8_linux_amd64.tar.gz) in `/usr/local/kubebuilder/bin` where they will be looked at by default.

#### Build and push your image
Replace `REGISTRY` with your Registry\`s URN:
```
$ cd $GOPATH/src/github.com/Dynatrace/dynatrace-operator
$ operator-sdk build REGISTRY/dynatrace-operator
$ docker push REGISTRY/dynatrace-operator
```

#### Deploy operator
Change the `image` field in `./deploy/operator.yaml` to the URN of your image.
Apart from that follow the instructions in the usage section above.