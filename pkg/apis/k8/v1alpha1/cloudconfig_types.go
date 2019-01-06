package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudConfigStatus defines the observed state of CloudConfig
type CloudConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudConfig is the Schema for the cloudconfigs API
// +k8s:openapi-gen=true
type CloudConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudConfigSpec   `json:"spec,omitempty"`
	Status CloudConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudConfigList contains a list of CloudConfig
type CloudConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudConfig{}, &CloudConfigList{})
}
