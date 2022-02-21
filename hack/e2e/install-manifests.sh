kubectl create ns dynatrace

kubectl apply -f config/deploy/kubernetes/kubernetes-all.yaml

kubectl -n dynatrace create secret generic dynakube --from-literal="apiToken=${APITOKEN}" --from-literal="paasToken=${PAASTOKEN}"

sleep 120
