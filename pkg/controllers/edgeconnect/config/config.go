package config

type EdgeConnect struct {
	// The technical identifier of the EdgeConnect.
	// This has to match the name that was specified in the configuration added in the app.
	Name string `yaml:"name"`

	// Your environment base URL.
	APIEndpointHost string `yaml:"api_endpoint_host"`

	// OAuth related section.
	OAuth OAuth `yaml:"oauth"`

	// Restricts outgoing HTTP requests to specified hosts.
	RestrictHostsTo []string `yaml:"restrict_hosts_to,omitempty"`

	// For communication over TLS-encrypted channels (HTTPS and secure WebSockets),
	// EdgeConnect verifies the identity of a host based on its certificate.
	RootCertificatePaths []string `yaml:"root_certificate_paths,omitempty"`

	// Proxy related section.
	Proxy Proxy `yaml:"proxy,omitempty"`

	// Secrets related section.
	Secrets []Secret `yaml:"secrets,omitempty"`
}

type OAuth struct {
	// The token endpoint URL of Dynatrace SSO.
	Endpoint string `yaml:"endpoint"`
	// The ID of the OAuth client that was created along with the EdgeConnect configuration.
	ClientID string `yaml:"client_id"`
	// The secret of the OAuth client that was created along with the EdgeConnect configuration.
	ClientSecret string `yaml:"client_secret"`
	// The URN identifying your tenant.
	Resource string `yaml:"resource"`
}

type Proxy struct {
	Auth       Auth   `yaml:"auth,omitempty"`
	Server     string `yaml:"server"`
	Exceptions string `yaml:"exceptions"`
	Port       uint32 `yaml:"port"`
}

type Auth struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Secret struct {
	Name            string   `yaml:"name"`
	Token           string   `yaml:"token"`
	FromFile        string   `yaml:"from_file"`
	RestrictHostsTo []string `yaml:"restrict_hosts_to,omitempty"`
}
