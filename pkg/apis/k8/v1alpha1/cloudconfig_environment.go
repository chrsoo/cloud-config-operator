package v1alpha1

import (
	"sync"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"time"
)

var client = http.Client{
	Timeout: time.Duration(10 * time.Second),
}

// Environment defines a CloudConfig environment configuration
type Environment struct {
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
	Credentials string `json:"credentials,omitempty"`

	// app spec file name, defaults to 'deployment.yaml'
	SpecFile string `json:"specFile,omitempty"`

	// application list property name
	AppList string `json:"appList,omitempty"`
}

func (env Environment) reconcile(wg *sync.WaitGroup) {
	defer wg.Done()
	env.ensureNamespace()

	// get the apps and order alphabetically to maintain consistency when applying the k8Config
	apps := env.getApps()
	sort.Strings(apps)

	// concatenate all app files into one configuration for the namespace
	config := make([]byte, 0, 1024)
	for _, app := range apps {
		file := env.getAppConfigFile(app, env.SpecFile)
		config = appendYAMLDoc(config, file)
	}
	env.apply(config)
}

// appendYAMLDoc appends a doc to an existing YAML configuration that is assumed to be valid YAML
func appendYAMLDoc(config, doc []byte) []byte {
	delim := []byte("---\n")
	doc = bytes.Trim(doc, "\n\r\t ")

	// don't append empty documents
	if len(doc) == 0 || bytes.HasSuffix(doc, []byte("---")) {
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

	if _, err := cmd.CombinedOutput(); err != nil {
		// TODO capture and log output

		// the namespace does not exist, create it!
		cmd = execCommand(
			"kubectl",
			"create",
			"namespace",
			env.Namespace)

		if _, err := cmd.CombinedOutput(); err != nil {
			// TODO capture and log output
			panic(err)
		}
	}
}

func (env Environment) apply(config []byte) {
	cmd := execCommand(
		"kubectl",
		"--namespace="+env.Namespace,
		"apply",
		"--prune",
		"--all=true",
		"-f",
		"-")
	cmd.Stdin = bytes.NewReader(config)
	if _, err := cmd.CombinedOutput(); err != nil {
		// TODO capture and log output
		panic(err)
	}
}

func (env Environment) getApps() []string {

	body := env.getAppConfig(env.AppName)
	if len(body) == 0 {
		// TODO Log warning as there is no config for the app
		return []string{}
	}

	// marshal the body as a map
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		// TODO log warning as the config cannot be parsed as a map
		return []string{}
	}

	if apps, exists := m[env.AppList]; exists {
		kind := reflect.ValueOf(apps).Kind()
		switch kind {
		case reflect.Slice:
			return interfaceToStringSlice(apps.([]interface{}))
		case reflect.String:
			// make sure we return a slice even if there is a single string value
			return []string{apps.(string)}
		default:
			// TODO Log warning as the AppList field points to a non valid list of apps
			return []string{}
		}
	} else {
		// TODO Log warning as there is no AppsList key in the map
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
	} else {
		baseURL = "http://" + env.Server
	}

	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return baseURL
}

// GetAppConfig retrieves the JSON configuration for the given path using the Spring Cloud Config URI pattern `/{app}/{profile}/{label}`
func (env Environment) getAppConfig(app string) []byte {
	url := env.baseURL() + app + "/" + strings.Join(env.Profile, ",") + "/" + env.Label
	return env.execute(http.MethodGet, url)
}

// GetAppConfigFile retrieves an arbitrary config file using the Spring Cloud Config URI pattern `/{app}/{profile}/{label}/{file}`
func (env Environment) getAppConfigFile(app string, file string) []byte {
	url := env.baseURL() + app + "/" + strings.Join(env.Profile, ",") + "/" + env.Label + "/" + file
	return env.execute(http.MethodGet, url)
}

func (env Environment) configureAuth(request *http.Request) {
	if env.Credentials == "" {
		return
	}

	if exists, err := testPath(env.Credentials); !exists || err != nil {
		// TODO log error
		if exists {
			panic(err)
		} else {
			panic("Credenitals directory '" + env.Credentials + "' does not exist!")
		}
	}

	// check if we have a token for bearer auth
	tokenPath := env.Credentials + "/token"
	if exists, _ := testPath(tokenPath); exists {
		token := readFile(tokenPath)
		request.Header.Set("Authorization", "Bearer "+token)
		return
	}

	// fall back to Basic Auth
	username := readFile(env.Credentials + "/username")
	password := readFile(env.Credentials + "/password")
	request.SetBasicAuth(username, password)
}

func (env Environment) execute(method string, url string) []byte {
	// TODO log url
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	env.configureAuth(request)

	// execute the request
	resp, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// read the body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// TODO handle IO errors properly
		return []byte{}
	}

	return body
}
