package startup

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type SecretConfig struct {
	MonitoringNodes map[string]string `json:"monitoringNodes"`
	// For the client
	ApiUrl      string `json:"apiUrl"`
	ApiToken    string `json:"apiToken"`
	PaasToken   string `json:"paasToken"`
	Proxy       string `json:"proxy"`
	NoProxy     string `json:"noProxy"`
	NetworkZone string `json:"networkZone"`

	// oneAgent
	OneAgentNoProxy string `json:"oneAgentNoProxy"`

	// For the injection
	TenantUUID          string `json:"tenantUUID"`
	HostGroup           string `json:"hostGroup"`
	InitialConnectRetry int    `json:"initialConnectRetry"`
	SkipCertCheck       bool   `json:"skipCertCheck"`

	HasHost           bool `json:"hasHost"`
	EnforcementMode   bool `json:"enforcementMode"`
	CSIMode           bool `json:"csiMode"`
	ReadOnlyCSIDriver bool `json:"readOnlyCSIDriver"`
}

func (secret SecretConfig) logContent() {
	const asterisks = "***"
	if secret.ApiToken != "" {
		secret.ApiToken = asterisks
	}

	if secret.PaasToken != "" {
		secret.PaasToken = asterisks
	}

	if secret.Proxy != "" {
		secret.Proxy = asterisks
	}

	log.Info("contents of secret config", "content", secret)
}

func newSecretConfigViaFs(fs afero.Fs) (*SecretConfig, error) {
	file, err := fs.Open(filepath.Join(consts.AgentConfigDirMount, consts.AgentInitSecretConfigField))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer file.Close()

	rawJson, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var config SecretConfig

	err = json.Unmarshal(rawJson, &config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Info("read secret from filesystem")
	config.logContent()

	return &config, nil
}

func newCertificatesViaFs(fs afero.Fs, filename string) (string, error) {
	filePem, err := fs.Open(filepath.Join(consts.AgentConfigDirMount, filename))
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("read certificates from filesystem - file not found", "filename", filename)

			return "", nil
		}

		return "", errors.WithStack(err)
	}
	defer filePem.Close()

	raw, err := io.ReadAll(filePem)
	if err != nil {
		return "", errors.WithStack(err)
	}

	log.Info("read certificates from filesystem", "content", "***", "filename", filename)

	return string(raw), nil
}
