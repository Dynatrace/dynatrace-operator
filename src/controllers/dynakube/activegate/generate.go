package activegate

import (
	"strings"
)

func (r *Reconciler) GenerateData() (map[string][]byte, error) {
	tenantInfo, err := r.dtc.GetTenantInfo()

	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		"uuid":                    []byte(tenantInfo.UUID),
		"token":                   []byte(tenantInfo.Token),
		"communication_endpoints": []byte(strings.Join(tenantInfo.Endpoints, ",")),
	}, nil
}
