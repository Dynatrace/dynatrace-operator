#!/bin/bash

echo "we build for arm too => combine images"
docker manifest create ${IMAGE_QUAY}:${{ needs.prepare.outputs.version }} ${IMAGE_QUAY}:${{ needs.prepare.outputs.version }}-arm64 ${IMAGE_QUAY}:${{ needs.prepare.outputs.version }}-amd64
docker manifest push ${IMAGE_QUAY}:${{ needs.prepare.outputs.version }}
