package proxy

// +kubebuilder:object:generate=true
type Spec struct {
	// Server address (hostname or IP address) of the proxy.
	Host string `json:"host,omitempty"`

	// NoProxy represents the NO_PROXY or no_proxy environment
	// variable. It specifies a string that contains comma-separated values
	// specifying hosts that should be excluded from proxying.
	NoProxy string `json:"noProxy,omitempty"`

	// Secret name which contains the username and password used for authentication with the proxy, using the
	// "Basic" HTTP authentication scheme.
	AuthRef string `json:"authRef,omitempty"`

	// Port of the proxy.
	Port uint32 `json:"port,omitempty"`
}
