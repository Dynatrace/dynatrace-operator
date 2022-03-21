kubectl kuttl test --config src/testing/e2e/kuttl/kuttl-test.yaml

# CLEAN-UP
kubectl delete dynakube --all -n dynatrace
kubectl -n dynatrace wait pod --for=delete -l app.kubernetes.io/component=oneagent --timeout=500s
kubectl delete -f config/deploy/kubernetes/kubernetes-all.yaml
kubectl delete namespace dynatrace
