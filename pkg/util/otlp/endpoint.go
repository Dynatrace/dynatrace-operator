package otlp

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
)

func GetOtlpIngestEndpoint(dk *dynakube.DynaKube) (string, error) {
	dtEndpoint := dk.APIURL() + "/v2/otlp"

	if dk.ActiveGate().IsEnabled() {
		tenantUUID, err := dk.TenantUUID()
		if err != nil {
			return "", err
		}

		serviceFQDN := capability.BuildServiceName(dk.Name) + "." + dk.Namespace + ".svc"

		dtEndpoint = fmt.Sprintf("https://%s/e/%s/api/v2/otlp", serviceFQDN, tenantUUID)
	}
	return dtEndpoint, nil
}
