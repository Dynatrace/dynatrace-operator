package dynakube

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/google/go-containerregistry/pkg/name"
)

const (
	errorUsingTenantImageAsCustom = `The 'image' field in the %s section can't be referencing an image directly from the Dynatrace Environment.`

	errorUnparsableImageRef = `The 'image' field in the %s section can't be parsed, make sure it's a valid image reference.`
)

func imageFieldHasTenantImage(_ context.Context, _ *dynakubeValidator, dk *dynakube.DynaKube) string {
	tenantHost := dk.ApiUrlHost()

	type imageField struct {
		value   string
		section string
	}

	imageFields := []imageField{
		{
			section: "ActiveGate",
			value:   dk.CustomActiveGateImage(),
		},
		{
			section: "OneAgent",
			value:   dk.CustomOneAgentImage(),
		},
	}

	messages := []string{}

	for _, field := range imageFields {
		message := checkImageField(field.value, field.section, tenantHost)
		if message != "" {
			messages = append(messages, message)
		}
	}

	return strings.Join(messages, ";")
}

func checkImageField(image, section, disallowedHost string) (errorMsg string) {
	if image != "" {
		ref, err := name.ParseReference(image)
		if err != nil {
			return fmt.Sprintf(errorUnparsableImageRef, section)
		}

		refUrl, _ := url.Parse(ref.Name())

		if refUrl.Host == disallowedHost {
			return fmt.Sprintf(errorUsingTenantImageAsCustom, section)
		}
	}

	return ""
}
