package oneagent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	"github.com/pkg/errors"
)

//
// TODO: the `arch` params should be removed and instead always the "github.com/Dynatrace/dynatrace-operator/pkg/arch" should be used

func (c *Client) Get(ctx context.Context, os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := getURL(os, installerType, flavor, arch, version, technologies, skipMetadata)
	sha256, err := c.makeRequestForBinary(ctx, url, writer)

	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "sha256", sha256)
	}

	return errors.WithStack(err)
}

// GetLatest gets the latest agent package for the given OS and installer type.
func (c *Client) GetLatest(ctx context.Context, os, installerType, flavor, arch string, technologies []string, skipMetadata bool, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := getLatestURL(os, installerType, flavor, arch, technologies, skipMetadata)
	sha256, err := c.makeRequestForBinary(ctx, url, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "sha256", sha256)
	}

	return errors.WithStack(err)
}

type VersionsResponse struct {
	AvailableVersions []string `json:"availableVersions"`
}

// GetVersions gets available agent versions for the given OS and installer type.
func (c *Client) GetVersions(ctx context.Context, os, installerType, flavor string) ([]string, error) {
	if len(os) == 0 || len(installerType) == 0 {
		return nil, errors.New("os or installerType is empty")
	}

	var resp VersionsResponse

	url := getVersionsURL(os, installerType)
	err := c.apiClient.GET(ctx, url).
		WithQueryParams(
			map[string]string{
				"flavor": flavor,
				"arch":   determineArch(installerType),
			},
		).
		WithPaasToken().
		Execute(&resp)

	return resp.AvailableVersions, errors.WithStack(err)
}

func (c *Client) GetViaInstallerURL(ctx context.Context, url string, writer io.Writer) error {
	sha256, err := c.makeRequestForBinary(ctx, url, writer)
	if err == nil {
		log.Info("downloaded agent file using given url", "url", url, "sha256", sha256)
	}

	return err
}

func (c *Client) makeRequestForBinary(ctx context.Context, url string, writer io.Writer) (string, error) {
	// Unsupported/Missing 'Accept' header.
	// need to add
	body, err := c.apiClient.GET(ctx, url).ExecuteRaw()
	if err != nil {
		return "", errors.WithStack(err)
	}

	hash := sha256.New()
	_, err = io.Copy(writer, io.TeeReader(bytes.NewReader(body), hash))

	return hex.EncodeToString(hash.Sum(nil)), err
}

func getURL(os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool) string {
	url := fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/version/%s?flavor=%s&arch=%s&bitness=64&skipMetadata=%t",
		os, installerType, version, flavor, arch, skipMetadata)

	return appendTechnologies(url, technologies)
}

func getLatestURL(os, installerType, flavor, arch string, technologies []string, skipMetadata bool) string {
	url := fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/latest?bitness=64&flavor=%s&arch=%s&skipMetadata=%t",
		os, installerType, flavor, arch, skipMetadata)

	return appendTechnologies(url, technologies)
}

func getVersionsURL(os, installerType string) string {
	return fmt.Sprintf("/v1/deployment/installer/agent/versions/%s/%s", os, installerType)
}

func appendTechnologies(url string, technologies []string) string {
	for _, tech := range technologies {
		url = fmt.Sprintf("%s&include=%s", url, tech)
	}

	return url
}

// determineArch gives you the proper arch value, because the OSAgent and ActiveGate images on the tenant-image-registry only have AMD images.
func determineArch(installerType string) string {
	if installerType == installer.TypeDefault {
		return ""
	}

	return arch.Arch
}
