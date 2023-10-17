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

import (
	"path/filepath"
)

const (
	DataPath       = "/data"
	DriverName     = "csi.oneagent.dynatrace.com"
	AgentBinaryDir = "bin"
	AgentRunDir    = "run"

	OverlayMappedDirPath = "mapped"
	OverlayVarDirPath    = "var"
	OverlayWorkDirPath   = "work"
	SharedAgentBinDir    = "codemodules"
	SharedAgentConfigDir = "config"

	DaemonSetName = "dynatrace-oneagent-csi-driver"

	UnixUmask = 0000
)

var MetadataAccessPath = filepath.Join(DataPath, "csi.db")

type CSIOptions struct {
	NodeId   string
	Endpoint string
	RootDir  string
}
