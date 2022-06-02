#!/bin/bash

echo "we build for amd only"
docker manifest create ${IMAGE_QUAY}:${{ needs.prepare.outputs.version }} ${IMAGE_QUAY}:${{ needs.prepare.outputs.version }}-amd64
docker manifest push ${IMAGE_QUAY}:${{ needs.prepare.outputs.version }}
