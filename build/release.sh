curl -s "https://raw.githubusercontent.com/ \
kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash

template_image="dynatrace-operator:snapshot"
current_image="dynatrace-operator:${TRAVIS_TAG}"
mkdir artefacts

kustomize build ./config/kubernetes -o kubernetes.yaml
kustomize build ./config/kubernetes-csi -o kubernetes-csi.yaml
kustomize build ./config/openshift -o openshift.yaml
kustomize build ./config/openshift-csi -o openshift-csi.yaml

sed "s/quay.io\/dynatrace\/${template_image}/docker.io\/dynatrace\/${current_image}/g" kubernetes.yaml >artefacts/kubernetes.yaml
sed "s/quay.io\/dynatrace\/${template_image}/docker.io\/dynatrace\/${current_image}/g" kubernetes-csi.yaml >artefacts/kubernetes-csi.yaml
sed "s/quay.io\/dynatrace\/${template_image}/registry.connect.redhat.com\/dynatrace\/${current_image}/g" openshift.yaml >artefacts/openshift.yaml
sed "s/quay.io\/dynatrace\/${template_image}/registry.connect.redhat.com\/dynatrace\/${current_image}/g" openshift-csi.yaml >artefacts/openshift-csi.yaml

cp ./config/samples/classicFullStack.yml artefacts/cr.yaml
