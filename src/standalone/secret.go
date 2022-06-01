package standalone

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

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
	if secret.ApiToken != "" {
		secret.ApiToken = "***"
	}
	if secret.PaasToken != "" {
		secret.PaasToken = "***"
	}
	if secret.TrustedCAs != "" {
		secret.TrustedCAs = "***"
	}
	if secret.TlsCert != "" {
		secret.TlsCert = "***"
	}
	log.Info("contents of secret config", "content", secret)
}

func newSecretConfigViaFs(fs afero.Fs) (*SecretConfig, error) {
	file, err := fs.Open(filepath.Join(ConfigDirMount, SecretConfigFieldName))
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
