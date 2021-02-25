/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
