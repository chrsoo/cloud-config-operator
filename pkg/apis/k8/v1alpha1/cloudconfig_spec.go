package v1alpha1

// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
import (
	"errors"
	"fmt"
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

// Init intializes the configuration for fist use
func (spec CloudConfigSpec) Init() {
	for key := range spec.Environments {
		env := spec.GetEnvironment(key)
		env.Configure()
	}
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
	fail := make(chan string, len(spec.Environments))
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
	close(fail)
	if failed := len(fail); failed > 0 {
		var failedEnvs []string
		for e := range fail {
			failedEnvs = append(failedEnvs, e)
		}
		msg := fmt.Sprintf("Reconcilation failed for %d out of %d environments", failed, len(spec.Environments))
		err := fmt.Errorf("Reconcilation failed for %v", failedEnvs)
		log.Error(err, msg)
	}
}

func (env Environment) finalize(wg *sync.WaitGroup, fail *chan string) {
	defer wg.Done()
	if err := recover(); err != nil {
		*fail <- env.Namespace
		switch err.(type) {
		case string:
			// handled errors
			log.Error(errors.New(err.(string)), "Recovered from a handled error", "namespace", env.Namespace)
		case error:
			// unahandled errors
			log.Error(err.(error), "Recovered from an unhandled error", "namespace", env.Namespace)
		default:
			// funky errors
			err := fmt.Errorf("[%T] %s", err, err)
			log.Error(err, "Recovered unknown error type", "namespace", env.Namespace)
		}
	}
}
