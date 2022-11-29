package standalone

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/spf13/afero"
)

type SecretConfig struct {
	// For the client
	ApiUrl        string `json:"apiUrl"`
	ApiToken      string `json:"apiToken"`
	PaasToken     string `json:"paasToken"`
	Proxy         string `json:"proxy"`
	NetworkZone   string `json:"networkZone"`
	TrustedCAs    string `json:"trustedCAs"`
	SkipCertCheck bool   `json:"skipCertCheck"`

	// For the injection
	TenantUUID          string            `json:"tenantUUID"`
	HasHost             bool              `json:"hasHost"`
	MonitoringNodes     map[string]string `json:"monitoringNodes"`
	TlsCert             string            `json:"tlsCert"`
	HostGroup           string            `json:"hostGroup"`
	InitialConnectRetry int               `json:"initialConnectRetry"`

	// For the enrichment
	ClusterID string `json:"clusterID"`
}

func (secret SecretConfig) logContent() {
	const asterisks = "***"
	if secret.ApiToken != "" {
		secret.ApiToken = asterisks
	}
	if secret.PaasToken != "" {
		secret.PaasToken = asterisks
	}
	if secret.TrustedCAs != "" {
		secret.TrustedCAs = asterisks
	}
	if secret.TlsCert != "" {
		secret.TlsCert = asterisks
	}
	log.Info("contents of secret config", "content", secret)
}

func newSecretConfigViaFs(fs afero.Fs) (*SecretConfig, error) {
	file, err := fs.Open(filepath.Join(config.AgentConfigDirMount, config.AgentInitSecretConfigField))
	if err != nil {
		return nil, err
	}

	rawJson, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config SecretConfig

	err = json.Unmarshal(rawJson, &config)
	if err != nil {
		return nil, err
	}

	log.Info("read secret from filesystem")
	config.logContent()

	return &config, nil
}
