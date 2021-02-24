package dtcsi

import "strings"

const DriverName = "csi.oneagent.dynatrace.com"

type CSIOptions struct {
	NodeID            string
	Endpoint          string
	SupportNamespaces map[string]bool
	DataDir           string
}

func ParseSupportNamespaces(sn string) map[string]bool {
	arr := strings.Split(sn, ",")

	snMap := make(map[string]bool, len(arr))
	for _, ns := range arr {
		snMap[ns] = true
	}

	return snMap
}
