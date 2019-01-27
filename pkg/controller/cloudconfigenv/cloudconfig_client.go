package cloudconfigenv

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("controller_cloudconfigenv")

// CloudConfigClient manages connections to a Spring Cloud Config Server
type CloudConfigClient struct {
	http     http.Client
	url      string
	insecure bool
	username string
	password string
	token    []byte
}

// Option type for the CloudConfigClient
type Option func(*CloudConfigClient)

// RootCA configures the CA certificate used to sign the server certificate
func RootCA(caCert []byte) Option {
	return func(c *CloudConfigClient) {
		if tr, ok := c.http.Transport.(*http.Transport); ok {
			// FIXME use the configured cert pool if it is not the default system pool!
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tr.TLSClientConfig.RootCAs = caCertPool
		} else {
			panic(fmt.Sprintf("Expected http.Transport for Root CA TLS configuration, found %T", c.http.Transport))
		}
	}
}

// TrustStore configures the default TLS certificate pool used by the client
func TrustStore(certs map[string][]byte) Option {
	return func(c *CloudConfigClient) {
		if tr, ok := c.http.Transport.(*http.Transport); ok {
			// FIXME use the configured cert pool if it is not the default system pool!
			caCertPool := x509.NewCertPool()
			for _, cert := range certs {
				caCertPool.AppendCertsFromPEM(cert)
			}
			tr.TLSClientConfig.RootCAs = caCertPool
		} else {
			panic(fmt.Sprintf("Expected http.Transport for TrustStore TLS configuration, found %T", c.http.Transport))
		}
	}
}

// BearerAuth configures the client to use Bearer Authentication with the provided token
func BearerAuth(token []byte) Option {
	return func(c *CloudConfigClient) {
		c.token = token
	}
}

// BasicAuth configures the client to use Basic Authentication with the given username and password
func BasicAuth(username, password string) Option {
	return func(c *CloudConfigClient) {
		c.username = username
		c.password = password
	}
}

// ClientCert side SSL configures the client to use Basic Authentication with the given username and password
func ClientCert(cert, key []byte) Option {

	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		panic("Could not load client certificcate: " + err.Error())
	}
	return func(c *CloudConfigClient) {
		if tr, ok := c.http.Transport.(*http.Transport); ok {
			tr.TLSClientConfig.Certificates = []tls.Certificate{tlsCert}
		} else {
			panic(fmt.Sprintf("Expected http.Transport for Client Cert configuration, found %T", c.http.Transport))
		}
	}
}

// Insecure allows use of URL's without SSL and disregards SSL validation errors
func Insecure() Option {
	return func(c *CloudConfigClient) {
		c.insecure = true
		if tr, ok := c.http.Transport.(*http.Transport); ok {
			tr.TLSClientConfig.InsecureSkipVerify = true
		} else {
			panic(fmt.Sprintf("Expected http.Transport for Insecure TLS configuration, found %T", c.http.Transport))
		}
	}
}

// Proxy defines a proxy URL for all HTTP requests
func Proxy(url *url.URL) Option {
	return func(c *CloudConfigClient) {
		if tr, ok := c.http.Transport.(*http.Transport); ok {
			tr.Proxy = http.ProxyURL(url)
		} else {
			panic(fmt.Sprintf("Expected http.Transport for Proxy configuration, found %T", c.http.Transport))
		}
	}
}

// Factory method to enable mocking the HTTP client in unit testing
var httpClientFactory = defaultHTTPClientFactory

func defaultHTTPClientFactory() *http.Client {
	tlsConfig := &tls.Config{}
	return &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
}

// New creates a new CloudConfigClient for the server URL and options
func New(url string, opts ...func(*CloudConfigClient)) (*CloudConfigClient, error) {
	c := &CloudConfigClient{
		http: *httpClientFactory(),
		url:  url,
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	if !strings.HasPrefix(url, "http") {
		if c.insecure {
			c.url = "http://" + url
		} else {
			c.url = "https://" + url
		}
	}

	if !strings.HasSuffix(url, "/") {
		c.url += "/"
	}

	// Validation
	if !c.insecure && strings.HasPrefix(c.url, "http://") {
		return nil, errors.New("Server URL must be secured unless the client is configured with the Insecure option")
	}
	// Initialize TLS if HTTP Transport is propertly condfigured
	if tr, ok := c.http.Transport.(*http.Transport); ok {
		tr.TLSClientConfig.BuildNameToCertificate()
	}

	return c, nil
}

// GetConfig returns the client config for the given app, label and profile.
func (client CloudConfigClient) GetConfig(app, label string, profile ...string) ([]byte, error) {
	if len(profile) > 0 {
		app += "-" + strings.Join(profile, ",")
	}
	url := client.url + label + "/" + app + ".json"
	return client.execute(http.MethodGet, url)
}

// GetConfigFile retrieves an arbitrary config file
func (client CloudConfigClient) GetConfigFile(file, app, label string, profile ...string) ([]byte, error) {
	url := client.url + app + "/" + strings.Join(profile, ",") + "/" + label + "/" + file
	return client.execute(http.MethodGet, url)
}

func (client CloudConfigClient) execute(method string, url string) ([]byte, error) {
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	log.Info(fmt.Sprintf("%s %s", method, request.URL.Path),
		"host", request.URL.Host,
		"method", request.Method,
		"scheme", request.URL.Scheme,
		"path", request.URL.Path,
		"port", request.URL.Port())

	client.configureAuth(request)
	// execute the request
	resp, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unhandled HTTP response '%s'", resp.Status)
	}

	// read the body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (client CloudConfigClient) getApps(list, app, label string, profile ...string) ([]string, error) {
	var body []byte
	body, err := client.GetConfig(app, label, profile...)
	if err != nil {
		return nil, err
	}

	// marshal the body as a map
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		err := fmt.Errorf("Could not marshal configuration as map")
		return nil, err
	}
	if apps, exists := m[list]; exists {
		kind := reflect.ValueOf(apps).Kind()
		switch kind {
		case reflect.Slice:
			return interfaceToStringSlice(apps.([]interface{})), nil
		case reflect.String:
			// make sure we return a slice even if there is a single string value
			return []string{apps.(string)}, nil
		default:
			err := fmt.Errorf("AppList field '%s' should be an array of strings, found %T", list, apps)
			return nil, err
		}
	}

	return nil, fmt.Errorf("AppList field '%s' does not exist in the '%s' configuration", list, app)
}

// Configure authentication and give priority to Bearer vs Basic auth
func (client CloudConfigClient) configureAuth(request *http.Request) {

	if len(client.token) > 0 {
		request.Header.Set("Authorization", "Bearer "+string(client.token))
		return
	}

	if client.username != "" {
		request.SetBasicAuth(client.username, client.password)
		return
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

// appendYAMLDoc appends a doc to an existing YAML configuration that is assumed to be valid YAML
func appendYAMLDoc(app string, config, doc []byte) []byte {
	delim := []byte("---\n")
	doc = bytes.Trim(doc, "\n\r\t ")

	// don't append empty documents
	if len(doc) == 0 || bytes.Equal(doc, []byte("---")) {
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
