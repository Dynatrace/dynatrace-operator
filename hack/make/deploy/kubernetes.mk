## Deploy the operator in the Kubernetes cluster configured in ~/.kube/config
deploy/kubernetes: manifests/crd/helm
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="kubernetes" \
			--set csidriver.enabled=true \
			--set manifests=true \
			--set image="$(BRANCH_IMAGE)" | kubectl apply -f -

## Deploy the operator in the Google Autopilot cluster configured in ~/.kube/config
deploy/gke-autopilot: manifests/crd/helm
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="gke-autopilot" \
			--set manifests=true \
			--set image="$(BRANCH_IMAGE)" | kubectl apply -f -

