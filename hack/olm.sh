#!/bin/bash

make bundle
cd ./config/olm/kubernetes/$(VERSION)
docker build -f bundle.Dockerfile . -t $(IMAGE)
docker push $(IMAGE)

opm index add --container-tool docker \
	--bundles $(IMAGE) \
	--tag $(IMG_INDEX)

docker push $(IMG_INDEX)
