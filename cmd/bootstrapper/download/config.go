package download

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
)

const (
	InputFileName = "dtclient.config"
)

type Config struct {
	URL      string `json:"url"`
	APIToken string `json:"apiToken"`

	Proxy       string `json:"proxy"`
	NoProxy     string `json:"noProxy"`
	NetworkZone string `json:"networkZone"`
	HostGroup   string `json:"hostGroup"`

	SkipCertCheck bool `json:"skipCertCheck"`
}

func (c Config) toDTClientOptionsV2() []dynatrace.OptionV2 {
	var options []dynatrace.OptionV2

	if c.APIToken != "" {
		options = append(options, dynatrace.WithAPIToken(c.APIToken))
		options = append(options, dynatrace.WithPaasToken(c.APIToken))
	}

	if c.HostGroup != "" {
		options = append(options, dynatrace.WithHostGroup(c.HostGroup))
	}

	if c.NetworkZone != "" {
		options = append(options, dynatrace.WithNetworkZone(c.NetworkZone))
	}

	if c.Proxy != "" {
		options = append(options, dynatrace.WithProxy(c.Proxy, c.NoProxy))
	}

	if c.SkipCertCheck {
		options = append(options, dynatrace.WithSkipCertificateValidation(c.SkipCertCheck))
	}

	return options
}

func configFromFs(inputDir string) (*Config, error) {
	inputFile := filepath.Join(inputDir, InputFileName)

	content, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, err
	}

	var config Config

	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
