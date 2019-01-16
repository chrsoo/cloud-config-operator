package v1alpha1

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
)

const (
	// SecretPathPrefix is the directory where secretes are mounted
	SecretPathPrefix = "/var/run/secret/cloud-config/"
)

// Environment defines a CloudConfig environment configuration
type Environment struct {
	httpClient http.Client

	// If Insecure is 'true' certificates are not required for
	// servers outside the cluster and SSL errors are ignored.
	Insecure bool `json:"insecure,omitempty"`

	// Key for the environment
	Key string `json:"key,omitempty"`

	// Name of the environment
	Name string `json:"name,omitempty"`

	// Namespace for all apps in this environment
	Namespace string `json:"namespace,omitempty"`

	// application name, defaults to the CloudConfig name
	AppName string `json:"appName,omitempty"`

	// List or profile names
	Profile []string `json:"profile,omitempty"`

	// label used for all apps, defaults to 'master'
	Label string `json:"label,omitempty"`

	// Cloud Config Server name or URL
	Server string `json:"server,omitempty"`

	// Cloud Config Server secret containing username and password
	// FIXME use a path, file or whatever
	Secret string `json:"secret,omitempty"`

	// app spec file name, defaults to 'deployment.yaml'
	SpecFile string `json:"specFile,omitempty"`

	// application list property name
	AppList string `json:"appList,omitempty"`
}

// Configure initializes the environment for fist use
func (env Environment) Configure() {
	// TODO make HTTP Timeout configuratble
	client := &http.Client{Timeout: time.Duration(10 * time.Second)}

	// TODO add proxy support and proxy authentication
	if env.Secret != "" {
		tlsConfig := &tls.Config{}
		env.configureTruststore(tlsConfig)
		env.configureSSLClientCert(tlsConfig)
		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		client.Transport = transport
		log.Info("Configured SSL from secret", "secret", env.Secret)
	}

	if env.Insecure {
		if tr, ok := client.Transport.(*http.Transport); ok {
			tr.TLSClientConfig.InsecureSkipVerify = true
		} else {
			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			client.Transport = &http.Transport{TLSClientConfig: tlsConfig}
		}
		log.Info("Skipping SSL verification!!!")
	}
	log.Info("Configured environment", "name", env.Name, "key", env.Key, "namespace", env.Namespace)
}

func (env Environment) configureSSLClientCert(tlsConfig *tls.Config) {
	certFile := env.getSecretPath() + "cert.pem"
	keyFile := env.getSecretPath() + "key.pem"

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

func (env Environment) configureTruststore(tlsConfig *tls.Config) {
	caFile := env.getSecretPath() + "ca.pem"
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

func (env Environment) reconcile() {
	// FIXME make sure that the check works for the DefaultClient
	if &env.httpClient == http.DefaultClient {
		err := errors.New("environment not initialized, call Init() before first use")
		panic(err)
	}
	env.ensureNamespace()

	// get the apps and order alphabetically to maintain consistency when applying the k8Config
	apps := env.getApps()
	sort.Strings(apps)

	// concatenate all app files into one configuration for the namespace
	spec := make([]byte, 0, 1024)
	for _, app := range apps {
		file := env.getAppConfigFile(app, env.SpecFile)
		// TODO ensure that the file is valid YAML before appending it to the spec
		spec = env.appendYAMLDoc(app, spec, file)
	}

	if len(spec) == 0 {
		log.Info(fmt.Sprintf("No specification to apply for namespace %s", env.Namespace),
			"namespace", env.Namespace)
	} else {
		env.apply(&spec)
	}
}

// appendYAMLDoc appends a doc to an existing YAML configuration that is assumed to be valid YAML
func (env Environment) appendYAMLDoc(app string, config, doc []byte) []byte {
	delim := []byte("---\n")
	doc = bytes.Trim(doc, "\n\r\t ")

	// don't append empty documents
	if len(doc) == 0 || bytes.Equal(doc, []byte("---")) {
		log.Info(fmt.Sprintf("Could not find deployment spec for '%s'", app),
			"namespace", env.Namespace,
			"app", app)
		return config
	}

	// add YAML document separator if doc is not already prefixed with separator
	if !bytes.HasPrefix(doc, delim) {
		config = append(config, delim...)
	}

	config = append(config, doc...)
	if !bytes.HasSuffix(config, []byte{'\n'}) {
		config = append(config, '\n')
	}

	return config
}

var execCommand = exec.Command

func (env Environment) ensureNamespace() {
	// Check if the namespace exists
	cmd := execCommand(
		"kubectl",
		"get",
		"namespace",
		env.Namespace)

	if msg, err := cmd.CombinedOutput(); err != nil {
		log.Info("Could not find namespace",
			"namespace", env.Namespace,
			"command", strings.Join(cmd.Args, " "),
			"output", string(msg))

		// the namespace does not exist, create it!
		cmd = execCommand(
			"kubectl",
			"create",
			"namespace",
			env.Namespace)

		log.Info("Creating namespace '"+env.Namespace+"'",
			"namespace", env.Namespace)

		if out, err := cmd.CombinedOutput(); err != nil {
			log.Error(err, "Could not create namespace",
				"namespace", env.Namespace,
				"command", strings.Join(cmd.Args, " "),
				"output", string(out))
			panic(err)
		}
	}
}

func (env Environment) apply(config *[]byte) {
	cmd := execCommand(
		"kubectl",
		"--namespace="+env.Namespace,
		"apply",
		"--prune",
		"--all",
		"-f",
		"-")
	cmd.Stdin = bytes.NewReader(*config)
	cmdString := strings.Join(cmd.Args, " ")

	log.Info(fmt.Sprintf("Reconciling environment '%s'", env.Namespace),
		"namespace", env.Namespace,
		"command", cmdString)

	var out []byte
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Error(err, fmt.Sprintf("Could not reconcile environment '%s'", env.Namespace),
			"namespace", env.Namespace,
			"command", cmdString,
			"output", string(out))
		panic(err)
	}

	log.Info(fmt.Sprintf("Reconciled environment '%s'", env.Namespace),
		"namespace", env.Namespace,
		"command", cmdString,
		"output", string(out))
}

func (env Environment) getApps() []string {

	body := env.getAppConfig(env.AppName)
	if len(body) == 0 {
		err := fmt.Errorf("App configuration for '%s' not found for %s", env.AppName, env.Key)
		log.Error(err, "Could not get the list of applications", "namespace", env.Namespace)
		return []string{}
	}

	// marshal the body as a map
	var m map[string]interface{}
	if err := yaml.Unmarshal(body, &m); err != nil {
		err := fmt.Errorf("Could not marshal configuration as map")
		log.Error(err, "Could not get the list of applications", "namespace", env.Namespace)
		return []string{}
	}

	if apps, exists := m[env.AppList]; exists {
		kind := reflect.ValueOf(apps).Kind()
		switch kind {
		case reflect.Slice:
			log.Info(fmt.Sprintf("Apps: %v", apps), "namespace", env.Namespace)
			return interfaceToStringSlice(apps.([]interface{}))
		case reflect.String:
			log.Info(fmt.Sprintf("Apps: [%v]", apps), "namespace", env.Namespace)
			// make sure we return a slice even if there is a single string value
			return []string{apps.(string)}
		default:
			err := fmt.Errorf("Value of AppList field '%s' should be a YAML list or string, found %T", env.AppList, apps)
			log.Error(err, "Could not get the list of applications", "namespace", env.Namespace)
			return []string{}
		}
	} else {
		err := fmt.Errorf("The AppList field '%s' does not exist in the app configuration for '%s'", env.AppList, env.AppName)
		log.Error(err, "Could not get the list of applications",
			"namespace", env.Namespace,
			"config", string(body))
		return []string{}
	}

}

// interfaceToStringSlice converts an interface slice to a string slice
func interfaceToStringSlice(interfaces []interface{}) []string {
	strings := make([]string, len(interfaces))
	for i, v := range interfaces {
		strings[i] = v.(string)
	}
	return strings
}

// baseURL returns the Environments Base URL based on the Server name
func (env Environment) baseURL() string {
	var baseURL string

	if strings.HasPrefix(env.Server, "http") {
		baseURL = env.Server
	} else if env.Insecure {
		baseURL = "http://" + env.Server
	} else {
		baseURL = "https://" + env.Server
	}

	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return baseURL
}

// GetAppConfig retrieves the JSON configuration for the given path using the Spring Cloud Config URI pattern `/{label}/{app}-{profile}.yml`
func (env Environment) getAppConfig(app string) []byte {
	url := env.baseURL() + env.Label + "/" + app + "-" + strings.Join(env.Profile, ",") + ".yaml"
	return env.execute(http.MethodGet, url)
}

// GetAppConfigFile retrieves an arbitrary config file using the Spring Cloud Config URI pattern `/{app}/{profile}/{label}/{file}`
func (env Environment) getAppConfigFile(app string, file string) []byte {
	url := env.baseURL() + app + "/" + strings.Join(env.Profile, ",") + "/" + env.Label + "/" + file
	return env.execute(http.MethodGet, url)
}

// getSecretPath returns a path to the secrets directory
func (env Environment) getSecretPath() string {
	// TODO switch to the paths package?

	// We assume the secret is a path if it contains at least one slash
	if strings.Contains(env.Secret, "/") {
		return strings.TrimRight(env.Secret, "/")
	}

	// Else we assume that the secret is a K8 secret mounted at the standard path
	return SecretPathPrefix + strings.Trim(env.Secret, "/")
}

func (env Environment) configureAuth(request *http.Request) {
	if env.Secret == "" {
		return
	}

	secretPath := env.getSecretPath()
	if !pathExists(secretPath) {
		panic("Credenitals directory '" + env.Secret + "' does not exist!")
	}

	// check if we have a token for bearer auth
	tokenPath := secretPath + "/token"
	if pathExists(tokenPath) {
		token := readFile(tokenPath)
		request.Header.Set("Authorization", "Bearer "+token)
		return
	}

	// fall back to Basic Auth
	username := readFile(secretPath + "/username")
	password := readFile(secretPath + "/password")
	request.SetBasicAuth(username, password)
}

func (env Environment) execute(method string, url string) []byte {
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}

	log.Info(fmt.Sprintf("%s %s", method, request.URL.Path),
		"host", request.URL.Host,
		"method", request.Method,
		"scheme", request.URL.Scheme,
		"path", request.URL.Path,
		"port", request.URL.Port())

	env.configureAuth(request)
	// execute the request
	resp, err := env.httpClient.Do(request)
	if err != nil {
		if _, ok := err.(*net.DNSError); ok {
			// Using a string indicates that we consider the error handled
			panic(err.Error)
		} else {
			// Using an error instance indicates that we consider the error unhandled
			panic(err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic("Unhandled HTTP response '" + resp.Status + "'")
	}

	// read the body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}
