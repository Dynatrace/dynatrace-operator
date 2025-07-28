// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha1
// +kubebuilder:validation:Optional
package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Spec defines the desired state of ActiveGate.
type Spec struct { //nolint:revive
	APIRUL string    `json:"apiURL,omitempty"`
	Proxy  ProxySpec `json:"proxy"`

	Group            string       `json:"group,omitempty"`
	CustomProperties *ValueSource `json:"customProperties,omitempty"`

	Template TemplateSpec `json:"template"`
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

// +kubebuilder:object:generate=true
type ValueSource struct {
	// Raw value for given property.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="property value",order=32,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// Name of the secret to get the property from.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="property secret name",order=33,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActiveGate is the Schema for the ActiveGate API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=edgeconnects,scope=Namespaced,categories=dynatrace,shortName={ec,ecs}
// +kubebuilder:printcolumn:name="ApiServer",type=string,JSONPath=`.spec.apiServer`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ActiveGate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Status Status `json:"status"`
	Spec   Spec   `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActiveGateList contains a list of ActiveGate
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
type ActiveGateList struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ActiveGate `json:"items"`
}

func init() {
	v1alpha1.SchemeBuilder.Register(&ActiveGate{}, &ActiveGateList{})
}
