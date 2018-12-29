package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudConfigServerSpec defines the desired state of CloudConfigServer
type CloudConfigServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// CloudConfigServerStatus defines the observed state of CloudConfigServer
type CloudConfigServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudConfigServer is the Schema for the cloudconfigservers API
// +k8s:openapi-gen=true
type CloudConfigServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudConfigServerSpec   `json:"spec,omitempty"`
	Status CloudConfigServerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudConfigServerList contains a list of CloudConfigServer
type CloudConfigServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudConfigServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudConfigServer{}, &CloudConfigServerList{})
}
