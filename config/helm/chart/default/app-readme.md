# Dynatrace Operator

The Dynatrace Operator supports rollout and lifecycle of various Dynatrace components in Kubernetes and OpenShift.

As of launch, the Dynatrace Operator can be used to deploy a containerized ActiveGate for Kubernetes API monitoring. New capabilities will be added to the Dynatrace Operator over time including metric routing, and API monitoring for AWS, Azure, GCP, and vSphere.

## Additional Instructions

Please make sure the CRD is applied before using this chart!

```
kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/dynatrace.com_dynakubes.yaml
```

To apply the CRD for Openshift follow the instructions in the [Github Repository](https://github.com/Dynatrace/helm-charts/tree/master/dynatrace-operator/chart/default#chart-installation).