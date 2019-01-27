package v1alpha1

import (
	"github.com/imdario/mergo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudConfigStatus defines the observed state of CloudConfig
type CloudConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	CronJobs []string `json:"cronJobs"`
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

// CloudConfigSpec defines the desired state of CloudConfig
type CloudConfigSpec struct {
	// Period is the number of seconds between each synchronization event
	Period int `json:"period,omitempty"`

	// Default environment properties
	CloudConfigEnvSpec `json:",omitempty"`

	// Environments where apps are managed
	Environments map[string]CloudConfigEnvSpec `json:"environments,omitempty"`
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

// GetEnvironment returns an environment from the spec falling back to default values for unspecified fields
func (config *CloudConfig) GetEnvironment(key string) *CloudConfigEnv {
	e, ok := config.Spec.Environments[key]
	if !ok {
		return nil
	}

	// use a copy as we do not want to corrupt the values supplied by the user
	env := e.DeepCopy()
	env.Env = key
	env.Sys = config.Name

	// env values that are already defined are retained
	mergo.Merge(env, config.Spec.CloudConfigEnvSpec)

	// apply default credential values
	mergo.Merge(env, DefaultCloudConfigEnvSpec(config.Spec.AppName, config.Name, key))

	// special rule for `Period` as ints are not merged
	if env.Period == 0 {
		env.Period = config.Spec.Period
	}

	fallBackIfEmpty(&env.AppName, config.Spec.AppName)
	fallBackIfEmpty(&env.AppName, config.Name)

	fallBackIfEmpty(&env.Label, config.Spec.Label)

	return &CloudConfigEnv{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.managedNamespaceForEnv(key),
			Namespace: config.Namespace,
			Labels: map[string]string{
				"app": env.AppName,
				"sys": env.Sys,
				"env": env.Env,
			},
		},
		Spec: *env,
	}
}

// IsManagedNamespace returns `true` if the given namespace is managed by one of the
// CloudConfig's environments.
func (config *CloudConfig) IsManagedNamespace(namespace string) bool {
	for key := range config.Spec.Environments {
		if namespace == config.managedNamespaceForEnv(key) {
			return true
		}
	}
	return false
}

func (config *CloudConfig) managedNamespaceForEnv(name string) string {
	return config.Name + "-" + name
}

func fallBackIfEmpty(field *string, defaultValue string) {
	if *field == "" {
		*field = defaultValue
	}
}

// DefaultCloudConfigEnvSpec returns the default spec for CloudConfigEnv
func DefaultCloudConfigEnvSpec(app, sys, env string) *CloudConfigEnvSpec {
	return &CloudConfigEnvSpec{
		AppName:     app,
		Sys:         sys,
		Env:         env,
		Label:       "master",
		Server:      "cloud-config-server:8888",
		Credentials: *DefaultCloudConfigCredentials(),
	}
}

// DefaultCloudConfigCredentials returns the default credentials
func DefaultCloudConfigCredentials() *CloudConfigCredentials {
	return &CloudConfigCredentials{
		Username: "username",
		Password: "password",
		Token:    "token",
		Cert:     "cert.pem",
		Key:      "cert.key",
		RootCA:   "ca.pem",
	}
}
