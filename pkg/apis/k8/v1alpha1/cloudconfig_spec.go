package v1alpha1

// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
import (
	"time"
	"crypto/x509"
	"io/ioutil"
	"crypto/tls"
	"net/http"
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

var httpClient http.Client

// Init intializes the configuration for fist use
func (spec CloudConfigSpec) Init() {
	// TODO make HTTP Timeout configuratble
	httpClient = http.Client{Timeout: time.Duration(10 * time.Second)}

	// TODO add proxy support and proxy authentication
	if spec.Secret != "" {
		tlsConfig := &tls.Config{}
		spec.configureTruststore(tlsConfig)
		spec.configureSSLClientCert(tlsConfig)
		tlsConfig.BuildNameToCertificate()
		httpClient.Transport = &http.Transport{TLSClientConfig: tlsConfig}
		log.Info("Configured SSL from secret", "secret", spec.Secret)
	}

	if spec.Insecure {
		if tr, ok := httpClient.Transport.(*http.Transport); ok {
			tr.TLSClientConfig.InsecureSkipVerify = true
		} else {
			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			httpClient.Transport = &http.Transport{TLSClientConfig: tlsConfig}
		}
		log.Info("Skipping SSL verification!!!")
	}
	log.Info("Configured Cloud Config", "name", spec.Name)
}

func (spec CloudConfigSpec) configureSSLClientCert(tlsConfig *tls.Config) {
	certFile := spec.getSecretPath() + "cert.pem"
	keyFile := spec.getSecretPath() + "key.pem"

	if !pathExists(certFile) && !pathExists(keyFile) {
		return
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic("Could not load client certificcate: " + err.Error())
	}
	// Add the client certificate
	tlsConfig.Certificates = []tls.Certificate{cert}
	// TODO log the name of the client as seen in the cert
	log.Info("Using SSL Client cerfificate from cert.pem and key.pem secrets")
}

func (spec CloudConfigSpec) configureTruststore(tlsConfig *tls.Config) {
	caFile := spec.getSecretPath() + "ca.pem"
	if !pathExists(caFile) {
		return
	}
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		panic("Could not load CA file: " + err.Error())
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig.RootCAs = caCertPool

	log.Info("Using CA from ca.pem secret")
}

// GetEnvironment returns an environment from the spec falling back to default values for unspecified fields
func (spec CloudConfigSpec) GetEnvironment(key string) *Environment {
	e := spec.Environments[key]
	env := e.DeepCopy() // use a copy as to no corrupt the values supplied by the user
	env.Name = key
	env.Namespace = spec.Name + "-" + key
	// env values that are already defined are retained
	mergo.Merge(env, spec.Environment)

	return env
}

// Reconcile the cloud configuration with the cluster state
func (spec CloudConfigSpec) Reconcile() {
	fail := make(chan string, len(spec.Environments))
	envs := make([]string, 0, len(spec.Environments))
	var wg sync.WaitGroup
	for key := range spec.Environments {
		env := spec.GetEnvironment(key)
		envs = append(envs, env.Name)
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
		msg := fmt.Sprintf("Reconcilation failed for %d out of %d environments of the '%s' Cloud Config App", failed, len(spec.Environments), spec.Name)
		err := fmt.Errorf("Reconcilation failed for %v", failedEnvs)
		log.Error(err, msg)
	} else {
		log.Info(fmt.Sprintf("Reconciled all %d environments of the '%s' Cloud Config App", len(spec.Environments), spec.Name))
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
