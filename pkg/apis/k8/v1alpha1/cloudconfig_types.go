package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudConfigEnv defines a CloudConfig environment configuration
type CloudConfigEnv struct {
	// Cloud Config name
	Name string `json:"name,omitempty"`

	// application name, defaults to the CloudConfig name
	AppName string `json:"appName,omitempty"`

	// List or profile names
	Profile []string `json:"profile,omitempty"`

	// label used for all apps, defaults to 'master'
	Label string `json:"label,omitempty"`

	// Cloud Config Server name or URL
	Server string `json:"server,omitempty"`

	// Cloud Config Server secret
	// FIXME use path file or whatever
	Credentials string `json:"credentials,omitempty"`

	// app spec file, defaults to 'deployment.yaml'
	// FIXME use path file or whatever
	SpecFile string `json:"specFile,omitempty"`

	// application list property name
	AppList string `json:"appList,omitempty"`

}

// CloudConfigSpec defines the desired state of CloudConfig
type CloudConfigSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Defaults CloudConfigEnv `json:"defaults,omitempty"`
	
	// Environments where apps are managed
	Environments map[string]CloudConfigEnv `json:"environments,omitempty"`
}

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
