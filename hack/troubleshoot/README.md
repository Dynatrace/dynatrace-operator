# Troubleshoot

Run this script while connected to the affected cluster.

## Scenarios

This script checks the following scenarios:

- Namespace
  - Namespace `dynatrace` exists (name overwrite-able via parameter)
- Dynakube
  - `CustomResourceDefinition` exists
  - `CustomResource` with the given name exists (name overwrite-able via parameter)
  - API url ends on `/api`
  - Secret with the same name as `dynakube` (or `.spec.tokens` if used) exists
  - Secret has `apiToken` set
  - Secret for `customPullSecret` exists if defined
- Tenant
  - Tenant is reachable from the operator pod using the same options as the `dynatrace-operator` (proxy, certificate, ...)
- Image (OneAgent and ActiveGate)
  - registry is accessible
  - image is accessible from the operator pod using registry from the tenant or (custom) pull secret
  
## Requirements

The script has the following dependencies:
- `bash`
- `kubectl` or `oc`
- `jq`
- `curl`

In addition, macOS has the following requirements:
- GNU implementation of `getopt`
  - please install gnu-based version and reopen terminal:
    ```bash
    brew install gnu-getopt && brew link --force gnu-getopt
    ```
- GNU implementation of `cut`
  - please install `gcut`: 
    ```bash
    brew install coreutils
    ```

## Usage

Run the following command to run the script.

```bash
bash -c "bash -c $(curl -fsSL https://raw.githubusercontent.com/dynatrace/dynatrace-operator/master/hack/troubleshoot/troubleshoot.sh)"
```

Make sure to inspect the contents of the troubleshooting script before executing it.

### Manual Instructions

1. Download `troubleshoot.sh` to the host with access to the affected cluster.
1. Make sure the file is executable: `chmod +x troubleshoot.sh`
1. Run script: `./troubleshoot.sh`

## Options

Specify options by appending them to the command, e.g: `./troubleshoot.sh --dynakube dynakube`

`-d DYNAKUKBE` or `--dynakube DYNAKUBE`
- allows checking a different dynakube object, by specifying its name
- default: `dynakube`

`-n NAMESPACE` or `--namespace NAMESPACE`
- allows specifying a different namespace
- default: `dynatrace`

`-c` or`--oc`
- changes CLI to `oc`
- default: `kubectl`
