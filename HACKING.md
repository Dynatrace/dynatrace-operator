# Dynatrace Operator

## Hack on Dynatrace Operator

[Operator SDK](https://github.com/operator-framework/operator-sdk) is the underlying framework for Dynatrace Operator. The `operator-sdk` tool needs to be installed upfront as outlined in the
[Operator SDK User Guide](https://sdk.operatorframework.io/docs/installation/).

### Installation

There are automatic builds from the main branch. The latest development build can be installed using `make deploy`.

#### Tests

The unit tests can be executed as follows:

```sh
make go/test
```
