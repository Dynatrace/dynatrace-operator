package dtversion

import (
	"context"
	"fmt"

	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/transports/alltransports"
)

func GetVersionLabel(imageName string, dockerConfig *DockerConfig) (string, error) {
	transportImageName := fmt.Sprintf("docker://%s", imageName)
	imageReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return "", err
	}

	systemContext := MakeSystemContext(imageReference.DockerReference(), dockerConfig)
	imageSource, err := imageReference.NewImageSource(context.TODO(), systemContext)
	if err != nil {
		return "", err
	}
	defer closeImageSource(imageSource)

	img, err := image.FromUnparsedImage(context.TODO(), systemContext, image.UnparsedInstance(imageSource, nil))
	if err != nil {
		return "", err
	}
	if img == nil {
		return "", fmt.Errorf("could not find image: '%s'", transportImageName)
	}

	inspectedImg, err := img.Inspect(context.TODO())
	if err != nil {
		return "", err
	}
	if inspectedImg == nil {
		return "", fmt.Errorf("could not inspect image: '%s'", transportImageName)
	}

	versionLabel, hasVersionLabel := inspectedImg.Labels[VersionKey]
	if !hasVersionLabel {
		return "", fmt.Errorf("remote does not have key '%s' in labels", VersionKey)
	}

	return versionLabel, nil
}
