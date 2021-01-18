curl -s "https://raw.githubusercontent.com/ \
kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash

template_image="dynatrace-operator:snapshot"
current_image="dynatrace-operator:${TRAVIS_TAG}"
mkdir artefacts

kustomize build ./config/manifests -o kubernetes.yaml
kustomize build ./config/manifests -o openshift.yaml

sed "s/quay.io\/dynatrace\/${template_image}/docker.io\/dynatrace\/${current_image}/g" kubernetes.yaml >artefacts/kubernetes.yaml
sed "s/quay.io\/dynatrace\/${template_image}/registry.connect.redhat.com\/dynatrace\/${current_image}/g" openshift.yaml >artefacts/openshift.yaml
