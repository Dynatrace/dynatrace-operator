curl -s "https://raw.githubusercontent.com/ \
kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash

template_image="dynatrace-operator:snapshot"
current_image="dynatrace-operator:${TRAVIS_TAG}"
mkdir artefacts

helm template dynatrace-operator ./config/helm/chart/default --namespace dynatrace --set platform="kubernetes" > kubernetes.yaml
sed -i '/app.kubernetes.io/d' kubernetes.yaml
sed -i '/helm.sh/d' kubernetes.yaml

helm template dynatrace-operator config/helm/chart/default --namespace dynatrace --set platform="openshift" > openshift.yaml
sed -i '/app.kubernetes.io/d' openshift.yaml
sed -i '/helm.sh/d' openshift.yaml

helm template dynatrace-operator config/helm/chart/default --namespace dynatrace --set platform="openshift-3-11" > openshift3.11.yaml
sed -i '/app.kubernetes.io/d' openshift3.11.yaml
sed -i '/helm.sh/d' openshift3.11.yaml

sed "s/quay.io\/dynatrace\/${template_image}/docker.io\/dynatrace\/${current_image}/g" kubernetes.yaml >artefacts/kubernetes.yaml
sed "s/quay.io\/dynatrace\/${template_image}/registry.connect.redhat.com\/dynatrace\/${current_image}/g" openshift.yaml >artefacts/openshift.yaml
sed "s/quay.io\/dynatrace\/${template_image}/registry.connect.redhat.com\/dynatrace\/${current_image}/g" openshift3.11.yaml >artefacts/openshift3.11.yaml

cp ./config/samples/cr.yaml artefacts/cr.yaml
