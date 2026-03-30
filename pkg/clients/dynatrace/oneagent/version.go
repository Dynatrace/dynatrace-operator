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

// Get gets the agent package for the given OS, installer type, flavor, arch and version.
func (c *Client) Get(ctx context.Context, os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	apiRequest := c.apiClient.GET(ctx, getURL(os, installerType, version)).
		WithQueryParams(map[string]string{
			"flavor":       flavor,
			"arch":         arch,
			"bitness":      "64",
			"skipMetadata": strconv.FormatBool(skipMetadata),
		}).
		WithRawQueryParams(technologiesQueryParams(technologies))

	sha256, err := makeRequestForBinary(apiRequest, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "sha256", sha256)
	}

	return errors.WithStack(err)
}

// GetLatest gets the latest agent package for the given OS, installer type, flavor and arch.
func (c *Client) GetLatest(ctx context.Context, os, installerType, flavor, arch string, technologies []string, skipMetadata bool, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	apiRequest := c.apiClient.GET(ctx, getLatestURL(os, installerType)).
		WithQueryParams(map[string]string{
			"flavor":       flavor,
			"arch":         arch,
			"bitness":      "64",
			"skipMetadata": strconv.FormatBool(skipMetadata),
		}).
		WithRawQueryParams(technologiesQueryParams(technologies))

	sha256, err := makeRequestForBinary(apiRequest, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "sha256", sha256)
	}

	return errors.WithStack(err)
}

type versionsResponse struct {
	AvailableVersions []string `json:"availableVersions"`
}

// GetVersions gets available agent versions for the given OS, installer type and flavor.
func (c *Client) GetVersions(ctx context.Context, os, installerType, flavor string) ([]string, error) {
	if len(os) == 0 || len(installerType) == 0 {
		return nil, errors.New("os or installerType is empty")
	}

	var resp versionsResponse

	params := map[string]string{
		"flavor": flavor,
	}

	oaArch := determineArch(installerType)
	if oaArch != "" {
		params["arch"] = oaArch
	}

	err := c.apiClient.GET(ctx, getVersionsURL(os, installerType)).
		WithQueryParams(params).
		WithPaasToken().
		Execute(&resp)

	return resp.AvailableVersions, errors.WithStack(err)
}

func (c *Client) GetViaInstallerURL(ctx context.Context, url string, writer io.Writer) error {
	apiRequest := c.apiClient.GET(ctx, url)

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
