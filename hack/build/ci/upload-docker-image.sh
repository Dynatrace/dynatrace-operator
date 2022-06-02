#!/bin/bash

# check if 2 arguments are set

docker load --input /tmp/operator-${{ inputs.platform }}.tar
docker tag ${SOURCE_IMAGE_TAG} ${TARGET_IMAGE_TAG}
docker push ${TARGET_IMAGE_TAG}
