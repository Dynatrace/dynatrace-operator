#!/bin/bash

if [[ "$GCR" == "true" ]]; then
    echo "$GCLOUD_SERVICE_KEY" | base64 -d | docker login -u _json_key --password-stdin https://gcr.io
    gcloud --quiet config set project "$GCP_PROJECT"
elif [[ "$IMAGE" != "$OAO_IMAGE_RHCC_SCAN" ]]; then
    TAG=$TAG-$TRAVIS_CPU_ARCH
fi

if [[ -z "$LABEL" ]]; then
    docker build . -f ./build/Dockerfile -t "$IMAGE:$TAG"
else
    docker build . -f ./build/Dockerfile -t "$IMAGE:$TAG" --label "$LABEL"
fi

echo "Pushing docker image"
docker push "$IMAGE:$TAG"
