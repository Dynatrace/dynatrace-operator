package standalone

import (
	"encoding/json"
	"path/filepath"

	"github.com/spf13/afero"
)

var (
	SecretConfigMount     = filepath.Join("mnt", "config")
	SecretConfigFieldName = "config"
)

const (
	ApiUrlFile        = "apiurl"
	ApiTokenFile      = "apitoken"
	PaasTokenFile     = "paastoken"
	ProxyFile         = "proxy"
	NetworkZoneFile   = "networkzone"
	TrustedCAsFile    = "trustedcas"
	SkipCertCheckFile = "skipcertcheck"
)

type SecretConfig struct {
	// For the client
	ApiUrl        string `json:"apiUrl"`
	ApiToken      string `json:"apiToken"`
	PaasToken     string `json:"passToken"`
	Proxy         string `json:"proxy"`
	NetworkZone   string `json:"networkZone"`
	TrustedCAs    string `json:"trustedCAs"`
	SkipCertCheck bool   `json:"skipCertCheck"`

	// For the injection
	TenantUUID      string            `json:"tenantUUID"`
	HasHost         bool              `json:"hasHost"`
	MonitoringNodes map[string]string `json:"monitoringNodes"`
	TlsCert         string            `json:"tlsCert"`
	HostGroup       string            `json:"hostGroup"`

	// For the enrichment
	ClusterID string `json:"clusterID"`
}

func newSecretConfigViaFs(fs afero.Fs) (*SecretConfig, error) {
	file, err := fs.Open(filepath.Join(SecretConfigMount, SecretConfigFieldName))
	if err != nil {
		return nil, err
	}
	var rawJson []byte
	_, err = file.Read(rawJson)
	if err != nil {
		return nil, err
	}
	var config *SecretConfig
	json.Unmarshal(rawJson, config)
	return config, nil
}
