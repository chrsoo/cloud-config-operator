package v1alpha1

// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
import (
	"sync"
    "github.com/imdario/mergo"
)

// CloudConfigSpec defines the desired state of CloudConfig
type CloudConfigSpec struct {

	// Default environment properties
	Environment `json:",omitempty"`

	// Environments where apps are managed
	Environments map[string]Environment `json:"environments,omitempty"`
}

// GetEnvironment returns an environment from the spec falling back to default values for unspecified fields
func (spec CloudConfigSpec) GetEnvironment(key string) *Environment {
	// TODO move spec fallbacks away from here
	// TODO default spec.Name to metadata.Name
	if spec.AppName == "" {
		spec.AppName = spec.Name
	}
	if spec.Key == "" {
		spec.Key = spec.Name
    }

	e := spec.Environments[key]
	env := e.DeepCopy()
	env.Key = key
	env.Namespace = spec.Key + "-" + key
	// default environment name to the key name
	if env.Name == "" {
		env.Name = env.Key
	}

	// env values that are already defined are retained
	mergo.Merge(env, spec.Environment)

	return env
}

// Reconcile the cloud configuration with the cluster state
func (spec CloudConfigSpec) Reconcile() {
	var wg sync.WaitGroup
	for key := range spec.Environments {
		env := spec.GetEnvironment(key)
		wg.Add(1)
		go env.reconcile(&wg)
	}
	wg.Wait()
}
