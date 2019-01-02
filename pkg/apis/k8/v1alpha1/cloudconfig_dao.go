package v1alpha1

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var client = http.Client{
	Timeout: time.Duration(10 * time.Second),
}

// ConfigServiceDAO returns raw config server resources
type ConfigServiceDAO interface {
	GetAppConfig(app string) []byte
	GetAppConfigFile(app, file string) []byte
}

// GetAppConfig retrieves the JSON configuration for the given path using the Spring Cloud Config URI pattern `/{app}/{profile}/{label}`
func (env Environment) GetAppConfig(app string) []byte {
	// create the request
	// FIXME verify JSON config URL!
	url := env.BaseURL() + app + "/" + strings.Join(env.Profile, ",") + "/" + env.Label
	return env.execute(http.MethodGet, url)
}

// GetAppConfigFile retrieves an arbitrary config file using the Spring Cloud Config URI pattern `/{app}/{profile}/{label}/{file}`
func (env Environment) GetAppConfigFile(app string, file string) []byte {
	// create the request
	// FIXME verify generic config file URL!
	url := env.BaseURL() + app + "/" + strings.Join(env.Profile, ",") + "/" + env.Label + "/" + file
	return env.execute(http.MethodGet, url)
}

func (env Environment) execute(method string, url string) []byte {
	// TODO log url
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		// TODO raise error
		return []byte{}
	}

	if env.Credentials != "" {
		request.SetBasicAuth(env.Username(), env.Password())
	}

	// execute the request
	resp, err := client.Do(request)
	if err != nil {
		// TODO handle error return codes properly
		return []byte{}
	}

	// read the body
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// TODO handle IO errors properly
		return []byte{}
	}

	return body
}
