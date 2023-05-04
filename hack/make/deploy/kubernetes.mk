## Deploy the operator in the Kubernetes cluster configured in ~/.kube/config
deploy/kubernetes: manifests/crd/helm
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="kubernetes" \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | kubectl apply -f -

## Deploy the operator in the Kubernetes cluster configured in ~/.kube/config
deploy/kubernetes-no-csi:
	ENABLE_CSI=false $(MAKE) deploy/kubernetes

## Deploy the operator in the Google Autopilot cluster configured in ~/.kube/config
deploy/gke-autopilot: manifests/crd/helm
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="gke-autopilot" \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | kubectl apply -f -

## Undeploy the operator in the Kubernetes cluster configured in ~/.kube/config
undeploy/kubernetes: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="kubernetes" \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | kubectl delete -f -

## Undeploy the operator in the Kubernetes cluster configured in ~/.kube/config
undeploy/kubernetes-no-csi:
	ENABLE_CSI=false $(MAKE) undeploy/kubernetes

## Undeploy the operator in the Google Autopilot cluster configured in ~/.kube/config
undeploy/gke-autopilot: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--set installCRD=true \
			--set platform="gke-autopilot" \
			--set manifests=true \
			--set image="$(IMAGE_URI)" | kubectl delete -f -
