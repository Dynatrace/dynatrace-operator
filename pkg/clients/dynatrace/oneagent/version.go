package oneagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	"github.com/pkg/errors"
)

type GetArgs struct {
	Os            string
	InstallerType string
	Flavor        string
	Arch          string
	Version       string
	Technologies  []string
	SkipMetadata  bool
}

// Get gets the agent package for the given OS, installer type, flavor, arch and version.
func (c *Client) Get(ctx context.Context, args GetArgs, writer io.Writer) error {
	if len(args.Os) == 0 || len(args.InstallerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	apiRequest := c.apiClient.GET(ctx, getURL(args.Os, args.InstallerType, args.Version)).
		WithPaasToken().
		WithQueryParams(map[string]string{
			"flavor":       args.Flavor,
			"arch":         args.Arch,
			"bitness":      "64",
			"skipMetadata": strconv.FormatBool(args.SkipMetadata),
		}).
		WithRawQueryParams(technologiesQueryParams(args.Technologies))

	sha256, err := makeRequestForBinary(apiRequest, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", args.Os, "type", args.InstallerType, "flavor", args.Flavor, "arch", args.Arch, "technologies", args.Technologies, "sha256", sha256)
	}

	return errors.WithStack(err)
}

type GetLatestArgs struct {
	Os            string
	InstallerType string
	Flavor        string
	Arch          string
	Technologies  []string
	SkipMetadata  bool
}

// GetLatest gets the latest agent package for the given OS, installer type, flavor and arch.
func (c *Client) GetLatest(ctx context.Context, args GetLatestArgs, writer io.Writer) error {
	if len(args.Os) == 0 || len(args.InstallerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	apiRequest := c.apiClient.GET(ctx, getLatestURL(args.Os, args.InstallerType)).
		WithPaasToken().
		WithQueryParams(map[string]string{
			"flavor":       args.Flavor,
			"arch":         args.Arch,
			"bitness":      "64",
			"skipMetadata": strconv.FormatBool(args.SkipMetadata),
		}).
		WithRawQueryParams(technologiesQueryParams(args.Technologies))

	sha256, err := makeRequestForBinary(apiRequest, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", args.Os, "type", args.InstallerType, "flavor", args.Flavor, "arch", args.Arch, "technologies", args.Technologies, "sha256", sha256)
	}

	return errors.WithStack(err)
}

type versionsResponse struct {
	AvailableVersions []string `json:"availableVersions"`
}

type GetVersionsArgs struct {
	Os            string
	InstallerType string
	Flavor        string
}

// GetVersions gets available agent versions for the given OS, installer type and flavor.
func (c *Client) GetVersions(ctx context.Context, args GetVersionsArgs) ([]string, error) {
	if len(args.Os) == 0 || len(args.InstallerType) == 0 {
		return nil, errors.New("os or installerType is empty")
	}

	var resp versionsResponse

	params := map[string]string{
		"flavor": args.Flavor,
	}

	oaArch := determineArch(args.InstallerType)
	if oaArch != "" {
		params["arch"] = oaArch
	}

	err := c.apiClient.GET(ctx, getVersionsURL(args.Os, args.InstallerType)).
		WithQueryParams(params).
		WithPaasToken().
		Execute(&resp)

	return resp.AvailableVersions, errors.WithStack(err)
}

func (c *Client) GetViaInstallerURL(ctx context.Context, url string, writer io.Writer) error {
	apiRequest := c.apiClient.GET(ctx, url).WithoutToken()

	sha256, err := makeRequestForBinary(apiRequest, writer)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Info("downloaded agent file using given url", "url", url, "sha256", sha256)

	return nil
}

func makeRequestForBinary(req core.APIRequest, writer io.Writer) (string, error) {
	hash := sha256.New()
	multiWriter := io.MultiWriter(writer, hash)

	err := req.
		WithHeader("Accept", "application/octet-stream").
		ExecuteWriter(multiWriter)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func getURL(os, installerType, version string) string {
	return fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/version/%s", os, installerType, version)
}

func getLatestURL(os, installerType string) string {
	return fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/latest", os, installerType)
}

func getVersionsURL(os, installerType string) string {
	return fmt.Sprintf("/v1/deployment/installer/agent/versions/%s/%s", os, installerType)
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
