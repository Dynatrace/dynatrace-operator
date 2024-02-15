package arch

import containerv1 "github.com/google/go-containerregistry/pkg/v1"

const (
	FlavorDefault     = "default"
	FlavorMultidistro = "multidistro"

	// These architectures are for the DynatraceAPI

	ArchX86   = "x86"
	ArchARM   = "arm"
	ArchPPCLE = "ppcle"
	ArchS390  = "s390"

	// These architectures are for the Image Registry

	AMDImageArch = "amd64"
	ARMImageArch = "arm64"

	DefaultImageOS = "linux"
)

var (
	ImagePlatform = containerv1.Platform{
		OS:           DefaultImageOS,
		Architecture: Arch,
	}
)
