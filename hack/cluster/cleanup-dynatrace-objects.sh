#!/bin/bash

# DESCRIPTION:
#   This script cleans up Dynatrace Operator related objects from a Kubernetes cluster.
#   It removes custom resources, the Dynatrace namespace, cluster-scoped resources,
#   and any related secrets and configmaps from all namespaces.
#
# USAGE:
#   ./cleanup-dynatrace-objects.sh [namespace]
#   If no namespace is provided, defaults to 'dynatrace'.

NAMESPACE="${1:-dynatrace}"
echo "Using namespace: $NAMESPACE"

echo -e "\nRemoving DynaKube and EdgeConnect custom resources"
kubectl delete dynakube --all -n "$NAMESPACE"
kubectl delete edgeconnect --all -n "$NAMESPACE"
kubectl -n "$NAMESPACE" wait pod --for=delete -l app.kubernetes.io/managed-by=dynatrace-operator --timeout=300s

echo -e "\nDeleting Dynatrace namespace"
kubectl delete namespace "$NAMESPACE" --ignore-not-found

echo -e "\nRemoving Dynatrace Operator cluster-scoped resources"
kubectl api-resources --verbs=list -o name --namespaced=false | \
    xargs -I {} sh -c \
    "kubectl delete {} --ignore-not-found -l app.kubernetes.io/name=dynatrace-operator 2>&1 | \
    grep -v 'No resources found'"

echo -n -e "\nRemoving Dynatrace Operator secrets and configmaps from all namespaces "
for ns in $(kubectl get ns -o jsonpath="{.items[*].metadata.name}"); do
    kubectl delete secret,cm -l app.kubernetes.io/name=dynatrace-operator --ignore-not-found -n "$ns" > /dev/null 2>&1
    printf '.'
done
echo "done"
