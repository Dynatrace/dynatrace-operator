// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtcsi

import (
	"os"

	"github.com/pkg/errors"
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
	SharedJobWorkDir     = "work"
	SharedAppMountsDir   = "appmounts"
	SharedDynaKubesDir   = "_dynakubes"
	SharedAgentConfigDir = "config"

	DaemonSetName = "dynatrace-oneagent-csi-driver"

	UnixUmask = 0000

	AppmountsDirPermissions = 0755
)

type CSIOptions struct {
	NodeID   string
	Endpoint string
	RootDir  string
}

func CreateDataPath() error {
	return errors.WithStack(os.MkdirAll(DataPath, os.ModePerm))
}
