# Scripts

## role-permissions2md - YAML to Markdown converter for operator permissions

This script can be used to create a table containing permissions needed by components of the Dynatrace Kubernetes Operator. The resulting tables can be used as input for our [public documentation](https://docs.dynatrace.com/docs/ingest-from/setup-on-k8s/reference/security#dto)

Pre-requisites:

- python3
- pyyaml

## Usage

It's best to use the `openshift-csi.yaml` manifest to cover all our needs. OpenShift needs the most permissions.

```sh
# local dev repo - direct call
# Create python virtual env
python3 -m venv venv
# activate virtual env
source venv/bin/activate
pip install pyyaml

make manifests
python3 <operator-repo>/hack/doc/role-permissions2md.py <operator-repo>/config/deploy/openshift/openshift-csi.yaml

# local dev repo - using make, result will be locally in permissions.md
make doc/permissions

# manifests from web
python3 <operator-repo>/hack/doc/role-permissions2md.py https://github.com/Dynatrace/dynatrace-operator/releases/download/v0.12.0/openshift.yaml
python3 <operator-repo>/hack/doc/role-permissions2md.py https://github.com/Dynatrace/dynatrace-operator/releases/download/v0.12.0/openshift-csi.yaml
```

## custom_resource_params_to_md.py

This script generates API docs for custom resources

Pre-requisites:

- python3
- pyyaml

## Usage

```bash
# local dev repo - direct call
# Create python virtual env
python3 -m venv venv
# activate virtual env
source venv/bin/activate
pip install pyyaml

python3 <operator-repo>/hack/doc/custom_resource_params_to_md.py './config/crd/bases/dynatrace.com_dynakubes.yaml'
```
