package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/google/go-containerregistry/pkg/name"
)

const (
	errorUnparsableImageRef = `Custom %s image can't be parsed, make sure it's a valid image reference.`
)

func imageFieldHasTenantImage(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	type imageField struct {
		value   string
		section string
	}

	imageFields := []imageField{
		{
			section: "ActiveGate",
			value:   dk.ActiveGate().GetCustomImage(),
		},
		{
			section: "OneAgent",
			value:   dk.OneAgent().GetCustomImage(),
		},
	}

	messages := []string{}

	for _, field := range imageFields {
		message := checkImageField(field.value, field.section)
		if message != "" {
			messages = append(messages, message)
		}
	}

	return strings.Join(messages, ";")
}

func checkImageField(image, section string) (errorMsg string) {
	if image != "" {
		_, err := name.ParseReference(image)
		if err != nil {
			return fmt.Sprintf(errorUnparsableImageRef, section)
		}
	}

	return ""
}
