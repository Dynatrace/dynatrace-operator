package activegate

import "github.com/pkg/errors"

const (
	CommunicationEndpointsName = "communication_endpoints"
	TenantTokenName            = "tenant-token"
	TenantUuidName             = "uuid"
)

func (r *Reconciler) GenerateData() (map[string][]byte, error) {
	tenantInfo, err := r.dtc.GetActiveGateTenantInfo(true)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		TenantUuidName:             []byte(tenantInfo.UUID),
		TenantTokenName:            []byte(tenantInfo.Token),
		CommunicationEndpointsName: []byte(tenantInfo.Endpoints),
	}, nil
}
