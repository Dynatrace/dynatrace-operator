#!/bin/bash

go test -cover -tags e2e,integration,containers_image_storage_stub ./...
