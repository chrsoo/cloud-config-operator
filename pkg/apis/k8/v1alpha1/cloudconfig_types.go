package v1alpha1

import (
    "strings"

    "github.com/imdario/mergo"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEnvironment returns an environment from the spec falling back to default
// values for unspecified fields
func (spec CloudConfigSpec) GetEnvironment(key string) *Environment {
    e := spec.Environments[key]
    env := e.DeepCopy()
    mergo.Merge(env, spec.Environment)
    return env
}

// BaseURL returns the Environments Base URL based on the Server name
func (env Environment) BaseURL() string {
    var baseURL string

    if strings.HasPrefix(env.Server, "http") {
        baseURL = env.Server
    } else {
        baseURL = "http://" + env.Server
    }

    if !strings.HasSuffix(baseURL, "/") {
        baseURL += "/"
    }

    return baseURL
}

// Username returns the username based on the given Credentials secret
func (env Environment) Username() string {
    // TODO parse the Credentials file and retrieve the username
    return "FIXME"
}

// Password returns the username based on the given Credentials secret
func (env Environment) Password() string {
    // TODO parse the Credentials file and retrieve the password
    return "FIXME"
}

// Environment defines a CloudConfig environment configuration
type Environment struct {
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

    // app spec file name, defaults to 'deployment.yaml'
    SpecFile string `json:"specFile,omitempty"`

    // application list property name
    AppList string `json:"appList,omitempty"`
}

// CloudConfigSpec defines the desired state of CloudConfig
type CloudConfigSpec struct {
    // Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
    Environment `json:",omitempty"`

    // Environments where apps are managed
    Environments map[string]Environment `json:"environments,omitempty"`
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
