package oneagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	goerrors "errors"
	"io"
	"net/url"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	"github.com/pkg/errors"
)

const agentDeploymentPath = "/v1/deployment/installer/agent"

var (
	errEmptyOS            = goerrors.New("OS is empty")
	errEmptyInstallerType = goerrors.New("installerType is empty")
)

type GetParams struct {
	OS            string
	InstallerType string
	Flavor        string
	Version       string
	Technologies  []string
	SkipMetadata  bool
}

// Get gets the agent package for the given OS, installer type, flavor, arch and version.
func (c *ClientImpl) Get(ctx context.Context, args GetParams, writer io.Writer) error {
	if len(args.OS) == 0 {
		return errEmptyOS
	}

	if len(args.InstallerType) == 0 {
		return errEmptyInstallerType
	}

	apiRequest := c.apiClient.GET(ctx, agentDeploymentPath).
		WithPath(args.OS, args.InstallerType, "version", args.Version).
		WithPaasToken().
		WithQueryParams(map[string]string{
			"flavor":       args.Flavor,
			"arch":         arch.Arch,
			"bitness":      "64",
			"skipMetadata": strconv.FormatBool(args.SkipMetadata),
		}).
		WithRawQueryParams(technologiesQueryParams(args.Technologies))

	sha256, err := makeRequestForBinary(apiRequest, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", args.OS, "type", args.InstallerType, "flavor", args.Flavor, "arch", arch.Arch, "technologies", args.Technologies, "sha256", sha256)
	}

	return errors.WithStack(err)
}

// GetLatest gets the latest agent package for the given OS, installer type, flavor and arch.
func (c *ClientImpl) GetLatest(ctx context.Context, args GetParams, writer io.Writer) error {
	if len(args.OS) == 0 {
		return errEmptyOS
	}

	if len(args.InstallerType) == 0 {
		return errEmptyInstallerType
	}

	apiRequest := c.apiClient.GET(ctx, agentDeploymentPath).
		WithPath(args.OS, args.InstallerType, "latest").
		WithPaasToken().
		WithQueryParams(map[string]string{
			"flavor":       args.Flavor,
			"arch":         arch.Arch,
			"bitness":      "64",
			"skipMetadata": strconv.FormatBool(args.SkipMetadata),
		}).
		WithRawQueryParams(technologiesQueryParams(args.Technologies))

	sha256, err := makeRequestForBinary(apiRequest, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", args.OS, "type", args.InstallerType, "flavor", args.Flavor, "arch", arch.Arch, "technologies", args.Technologies, "sha256", sha256)
	}

	return errors.WithStack(err)
}

type versionsResponse struct {
	AvailableVersions []string `json:"availableVersions"`
}

// GetVersions gets available agent versions for the given OS, installer type and flavor.
func (c *ClientImpl) GetVersions(ctx context.Context, args GetParams) ([]string, error) {
	if len(args.OS) == 0 {
		return nil, errEmptyOS
	}

	if len(args.InstallerType) == 0 {
		return nil, errEmptyInstallerType
	}

	var resp versionsResponse

	params := map[string]string{
		"flavor": args.Flavor,
	}

	oaArch := determineArch(args.InstallerType)
	if oaArch != "" {
		params["arch"] = oaArch
	}

	err := c.apiClient.GET(ctx, agentDeploymentPath).
		WithPath("versions", args.OS, args.InstallerType).
		WithQueryParams(params).
		WithPaasToken().
		Execute(&resp)

	return resp.AvailableVersions, errors.WithStack(err)
}

func makeRequestForBinary(req core.Request, writer io.Writer) (string, error) {
	hash := sha256.New()
	multiWriter := io.MultiWriter(writer, hash)

	_, err := req.
		WithHeader("Accept", "application/octet-stream").
		ExecuteWriter(multiWriter)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func technologiesQueryParams(technologies []string) url.Values {
	params := make(url.Values)
	for _, tech := range technologies {
		params.Add("include", tech)
	}

	return params
}

// determineArch gives you the proper arch value, because the OSAgent and ActiveGate images on the tenant-image-registry only have AMD images.
func determineArch(installerType string) string {
	if installerType == installer.TypeDefault {
		return ""
	}

	return arch.Arch
}
