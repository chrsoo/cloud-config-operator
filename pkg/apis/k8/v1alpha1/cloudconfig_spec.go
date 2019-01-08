package v1alpha1

// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
import (
	"sync"
    "github.com/imdario/mergo"
)

// CloudConfigSpec defines the desired state of CloudConfig
type CloudConfigSpec struct {
	// Schedule is the cron job schedule to used for reconciling the Cloud Config configuration
	Schedule string `json:"schedule,omitempty"`

	// Default environment properties
	Environment `json:",omitempty"`

	// Environments where apps are managed
	Environments map[string]Environment `json:"environments,omitempty"`
}

// GetEnvironment returns an environment from the spec falling back to default values for unspecified fields
func (spec CloudConfigSpec) GetEnvironment(key string) *Environment {
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
	fail := make(chan bool, len(spec.Environments))
	var wg sync.WaitGroup
	for key := range spec.Environments {
		env := spec.GetEnvironment(key)
		wg.Add(1)
		go func() {
			defer env.finalize(&wg, &fail)
			env.reconcile()
		}()
	}

	wg.Wait()
	if len(fail) > 0 {
		panic("Reconcilation failed")
	}
}

func (env Environment) finalize(wg *sync.WaitGroup, fail *chan bool) {
	defer wg.Done()
	if err := recover(); err != nil {
		switch err.(type) {
			case string:
				log.Info(err.(string), "namespace", env.Namespace)
			case error:
				log.Error(err.(error), "namespace", env.Namespace)
				*fail <- true
			default:
				panic(err)
		}
	}
}
