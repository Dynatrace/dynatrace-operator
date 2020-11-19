# Dynatrace Operator

## Hack on Dynatrace Operator

[Operator SDK](https://github.com/operator-framework/operator-sdk) is the underlying framework for Dynatrace Operator. The `operator-sdk` tool needs to be installed upfront as outlined in the
[Operator SDK User Guide](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#install-the-operator-sdk-cli).

### Installation

There are automatic builds from the master branch. The latest development build can be installed as follows:

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

The unit tests can be executed as follows:

```
$ go test ./...
```

#### Build and push your image

Replace `REGISTRY` with your Registry\`s URN:
```
$ cd $GOPATH/src/github.com/Dynatrace/dynatrace-operator
$ operator-sdk build REGISTRY/dynatrace-operator
$ docker push REGISTRY/dynatrace-operator
```

#### Deploy operator

Change the `image` field in `./deploy/manifest/deployment-operator.yaml` to the URN of your image.
Apart from that follow the instructions in the usage section above.
