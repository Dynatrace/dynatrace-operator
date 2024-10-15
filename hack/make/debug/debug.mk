
## Run the operator locally
debug/operator:
	kubectl -n dynatrace scale --replicas=0 deployment/dynatrace-operator
	POD_NAMESPACE=dynatrace RUN_LOCAL=true go run ./cmd operator

## Run the webhook locally (requires running telepresence)
debug/webhook:
	env $$(cat local/telepresence.env | xargs) go run ./cmd webhook-server --certs-dir=./local/certs/

## In case of code changes, closes the tunnel, rebuilds/deploys the image and opens the tunnel again.
debug/csi/redeploy: debug/tunnel/stop debug/build
	kd delete pod -l internal.oneagent.dynatrace.com/app=csi-driver # Delete the pod to force a redownload of the image
	make debug/tunnel/start

## Build image with Delve debugger included.
debug/build:
	DEBUG=true make images/build/push

## Install image with necessary changes to deployments.
debug/deploy:
	DEBUG=true make deploy/helm

## Install and setup Telepresence to intercept requests to the webhook
debug/telepresence/install:
	telepresence helm install --create-namespace
	telepresence connect -n dynatrace
	telepresence intercept dynatrace-webhook -p 8443 --env-file local/telepresence.env

## Stop Telepresence and remove all changes made to the cluster.
debug/telepresence/uninstall:
	telepresence quit
	telepresence helm uninstall

## Opens a tunnel from your local machine to the CSI driver pod
debug/tunnel/start:
	kubectl port-forward -n dynatrace $$(kubectl get pod -n dynatrace -l app.kubernetes.io/component=csi-driver -o jsonpath='{.items[0].metadata.name}') 40000:40000 40001:40001 & echo $$! > /tmp/csi-driver-port-forward.pid

## Stop the tunnel from local machine to CSI driver pod.
debug/tunnel/stop:
	@if [ -f /tmp/csi-driver-port-forward.pid ]; then \
		kill $$(cat /tmp/csi-driver-port-forward.pid) && rm /tmp/csi-driver-port-forward.pid; \
	else \
		echo "No port-forward process found."; \
	fi
