
## Run the operator locally
debug/operator:
	kubectl -n dynatrace scale --replicas=0 deployment/dynatrace-operator
	POD_NAMESPACE=dynatrace RUN_LOCAL=true go run ./cmd operator

## Run the webhook locally (requires running telepresence)
debug/webhook:
	env $$(cat local/telepresence.env | xargs) go run ./cmd webhook-server --certs-dir=./local/certs/

## Build image with debugging information and dlv debugger (required for CSI debugging)
debug/build:
	DEBUG=true make images/build/push

## Deploy the operator with modified csi-driver and webhook (requires image built with debug information)
debug/deploy:
	DEBUG=true make deploy/helm

## install and start telepresence in the cluster
debug/telepresence/install:
	telepresence helm install --create-namespace
	telepresence connect -n dynatrace
	telepresence intercept dynatrace-webhook -p 8443 --env-file local/telepresence.env

## stop telepresence
debug/telepresence/stop:
	telepresence quit
	telepresence helm uninstall

## Forwards the CSI server ports to localhost:40000 and localhost:40001 (required for debugging the CSI driver)
debug/tunnel:
	kubectl port-forward -n dynatrace $$(kubectl get pod -n dynatrace -l app.kubernetes.io/component=csi-driver -o jsonpath='{.items[0].metadata.name}') 40000:40000 40001:40001 & echo $$! > /tmp/csi-driver-port-forward.pid

## Cancels the port-forward
debug/tunnel/stop:
	@if [ -f /tmp/csi-driver-port-forward.pid ]; then \
		kill $$(cat /tmp/csi-driver-port-forward.pid) && rm /tmp/csi-driver-port-forward.pid; \
	else \
		echo "No port-forward process found."; \
	fi
