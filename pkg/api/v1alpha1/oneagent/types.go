// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha1
// +kubebuilder:validation:Optional
package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Spec defines the desired state of ActiveGate.
type Spec struct { //nolint:revive
	APIRUL        string    `json:"apiURL,omitempty"`
	Tokens        string    `json:"tokens,omitempty"`
	TrustedCAs    string    `json:"trustedCAs,omitempty"`
	NetworkZone   string    `json:"networkZone,omitempty"`
	Proxy         ProxySpec `json:"proxy"`
	SkipCertCheck bool      `json:"skipCertCheck,omitempty"`

	InitialConnectRetry *int `json:"initialConnectRetry,omitempty"`

	HostGroup string `json:"hostGroup,omitempty"`

	HostAgent HostAgentSpec `json:"hostAgent"`
	// TODO: maybe add logmon here?
	CodeModule CodeModulesSpec `json:"codeModule"`
}

type ProxySpec struct {
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

	Propagate *bool `json:"propagate,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneAgent is the Schema for the OneAgent API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=oneagents,scope=Namespaced,categories=dynatrace,shortName={oa,oas}
// +kubebuilder:printcolumn:name="ApiServer",type=string,JSONPath=`.spec.apiURL`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OneAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Status Status `json:"status"`
	Spec   Spec   `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneAgentList contains a list of ActiveGate
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
type OneAgentList struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []OneAgent `json:"items"`
}

func init() {
	v1alpha1.SchemeBuilder.Register(&OneAgent{}, &OneAgentList{})
}
