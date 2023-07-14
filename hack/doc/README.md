# role-permissions2md - YAML to Markdown converter for operator permissions

This script can be used to create a table containing permissions needed by components of the Dynatrace Kubernetes Operator. The resulting tables can be used as input for SUS tickets to update our public documentation here: https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dt-component-permissions#dto

## Usage
```
# local dev repo
make manifests/kubernetes
python3 role-permissions2md.py <operator-repo>/config/deploy/kubernetes/kubernetes-all.yaml

# manifests from web
python3 role-permissions2md.py https://github.com/Dynatrace/dynatrace-operator/releases/download/v0.12.0/kubernetes.yaml
python3 role-permissions2md.py https://github.com/Dynatrace/dynatrace-operator/releases/download/v0.12.0/kubernetes-csi.yaml
```
