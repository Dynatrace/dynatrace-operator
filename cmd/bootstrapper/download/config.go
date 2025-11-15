package download

import (
	"encoding/json"
	"os"
	"path/filepath"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
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

func (c Config) toDTClientOptions() []dtclient.Option {
	options := []dtclient.Option{}

	if c.HostGroup != "" {
		options = append(options, dtclient.HostGroup(c.HostGroup))
	}

	if c.NetworkZone != "" {
		options = append(options, dtclient.NetworkZone(c.NetworkZone))
	}

	if c.Proxy != "" {
		options = append(options, dtclient.Proxy(c.Proxy, c.NoProxy))
	}

	if c.SkipCertCheck {
		options = append(options, dtclient.SkipCertificateValidation(c.SkipCertCheck))
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
