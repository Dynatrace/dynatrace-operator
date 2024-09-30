## Prepares all necessary CSI server resources for debugging (e.g. set limits, change build flags and startup command, ...)
debug/prepare/csi-server:
	git apply hack/make/debug/csi-server.patch

## Prepares all necessary CSI provisioner resources for debugging (e.g. set limits, change build flags and startup command, ...)
debug/prepare/csi-provisioner:
	git apply hack/make/debug/csi-provisioner.patch

## Forwards the CSI server port to localhost:40000
debug/tunnel/csi-driver:
	kubectl port-forward -n dynatrace $$(kubectl get pod -n dynatrace -l app.kubernetes.io/component=csi-driver -o jsonpath='{.items[0].metadata.name}') 40000:40000

## Remove all changes made by 'debug/prepare/csi-server'
debug/remove/csi-server:
	git apply -R hack/make/debug/csi-server.patch

## Remove all changes made by 'debug/prepare/csi-provisioner'
debug/remove/csi-provisioner:
	git apply -R hack/make/debug/csi-provisioner.patch

## Run the operator locally
debug/operator:
	kubectl -n dynatrace scale --replicas=0 deployment/dynatrace-operator
	POD_NAMESPACE=dynatrace RUN_LOCAL=true go run ./cmd operator
