package edgeconnect

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/api"
)

const defaultEdgeConnectRepository = "docker.io/dynatrace/edgeconnect"

func (edgeConnect *EdgeConnect) Image() string {
	repository := defaultEdgeConnectRepository
	tag := api.LatestTag

	if edgeConnect.Spec.ImageRef.Repository != "" {
		repository = edgeConnect.Spec.ImageRef.Repository
	}
	if edgeConnect.Spec.ImageRef.Tag != "" {
		tag = edgeConnect.Spec.ImageRef.Tag
	}

	return fmt.Sprintf("%s:%s", repository, tag)
}

func (edgeConnect *EdgeConnect) IsCustomImage() bool {
	return edgeConnect.Spec.ImageRef.Repository != ""
}
