package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const warningDeprecatedVersion = `version field is deprecated. Please use "%s" field instead to set a version.`

func deprecatedOneAgentVersionField(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	oa := dk.OneAgent()

	if oa.GetCustomVersion() != "" {
		switch {
		case oa.IsApplicationMonitoringMode():
			return fmt.Sprintf(warningDeprecatedVersion, "codeModulesImage")
		case oa.IsCloudNativeFullstackMode():
			return fmt.Sprintf(warningDeprecatedVersion, "image and/or codeModulesImage")
		default:
			return fmt.Sprintf(warningDeprecatedVersion, "image")
		}
	}

	return ""
}
