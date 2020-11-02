package dao

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/version"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

func GetImageLabels(imageName string, dockerConfig *parser.DockerConfig) (map[string]string, error) {
	transportImageName := fmt.Sprintf("docker://%s", imageName)
	imageReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return nil, err
	}

	systemContext := version.MakeSystemContext(imageReference.DockerReference(), dockerConfig)
	imageSource, err := imageReference.NewImageSource(context.TODO(), systemContext)
	if err != nil {
		return nil, err
	}
	defer closeImageSource(imageSource)

	img, err := image.FromUnparsedImage(context.TODO(), systemContext, image.UnparsedInstance(imageSource, nil))
	if err != nil {
		return nil, err
	}
	if img == nil {
		return nil, fmt.Errorf("could not find image: '%s'", transportImageName)
	}

	inspectedImg, err := img.Inspect(context.TODO())
	if err != nil {
		return nil, err
	}
	if inspectedImg == nil {
		return nil, fmt.Errorf("could not inspect image: '%s'", transportImageName)
	}

	return inspectedImg.Labels, nil
}

func closeImageSource(source types.ImageSource) {
	if source != nil {
		// Swallow error
		_ = source.Close()
	}
}
