## Deploys the operator using a snapshot deployer image for a standard GKE cluster
deploy/gke/deployer:
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/application/master/deploy/kube-app-manager-aio.yaml
	./hack/gcr/deploy.sh ":snapshot-${SNAPSHOT_SUFFIX}"

## Deploys the operator using a snapshot deployer image for an autopilot GKE cluster
deploy/autopilot/deployer:
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/application/master/deploy/kube-app-manager-aio.yaml
	./hack/gcr/deploy.sh ":snapshot-${SNAPSHOT_SUFFIX}" "gke-autopilot"
