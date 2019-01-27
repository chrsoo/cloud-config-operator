package cloudconfigenv

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestDefaultOptions(t *testing.T) {
	client, err := New("https://test.com/")
	assert.Nil(t, err)
	assert.Equal(t, "https://test.com/", client.url)
	tr := client.http.Transport.(*http.Transport)
	assert.False(t, tr.TLSClientConfig.InsecureSkipVerify)

	client, err = New("https://test.com")
	assert.Equal(t, "https://test.com/", client.url, "Client URL should end with slash")

	client, err = New("test.com")
	assert.Equal(t, "https://test.com/", client.url, "Secure client URL should start with https")

	client, err = New("http://test.com")
	assert.Error(t, err, "Using an insecure URL should result in an error if client is not Insecure")
	assert.Nil(t, client, "Client should not be returned if there is an error")
}

func TestInsecure(t *testing.T) {
	client, err := New("test.com", Insecure())
	assert.Nil(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "http://test.com/", client.url, "Insecure client URL should start with http if unspecified")

	tr := client.http.Transport.(*http.Transport)
	assert.True(t, tr.TLSClientConfig.InsecureSkipVerify)

	client, err = New("https://test.com", Insecure())
	assert.Equal(t, "https://test.com/", client.url, "Secure client URLs should not be changed when client is insecure")
}

func TestProxyOption(t *testing.T) {
	u, err := url.Parse("http://localhost:3128")
	assert.NoError(t, err)
	New(TestBaseURL, Proxy(u))
	// TODO assert that HTTP requests connect to the proxy
}

func TestTrustStoreOption(t *testing.T) {
	// we use the client.pem just to have more than one test cert!
	certs := map[string][]byte{
		"ca.pem":     []byte(testRootCAPem),
		"client.pem": []byte(testClientPem),
	}

	client, err := New(TestBaseURL, TrustStore(certs))
	assert.NoError(t, err)
	assert.NotNil(t, client)
	// TODO assert that the certs are configured in the client tlsConfig's truststore!
}
func TestRootCAOption(t *testing.T) {
	client, err := New(TestBaseURL, RootCA([]byte(testRootCAPem)))
	assert.NoError(t, err)
	assert.NotNil(t, client)
	// TODO assert that the rootCA cert is configured in the client tlsConfig's truststore!
}

func TestClientCertOption(t *testing.T) {
	client, err := New(TestBaseURL, ClientCert([]byte(testClientPem), []byte(testClientKey)))
	assert.NoError(t, err)
	assert.NotNil(t, client)
	// TODO assert that the client certificate is installed!
}

func TestBearerAuthOption(t *testing.T) {
	token := []byte("TOKEN_VALUE")
	client, err := New(TestBaseURL, BearerAuth(token))
	assert.NoError(t, err)
	assert.Equal(t, "TOKEN_VALUE", string(client.token), "Token not configured")

	request, err := http.NewRequest("GET", "https://test.com", nil)
	assert.NoError(t, err)
	client.configureAuth(request)
	assert.Equal(t, "Bearer TOKEN_VALUE", request.Header.Get("Authorization"), "Bearer Auth not properly configured")

	client, err = New(TestBaseURL, BearerAuth(token), BasicAuth("username", "password"))
	assert.NoError(t, err)

	request, err = http.NewRequest("GET", "https://test.com", nil)
	assert.NoError(t, err)
	client.configureAuth(request)
	assert.Equal(t, "Bearer TOKEN_VALUE", request.Header.Get("Authorization"), "A Bearer Auth should override Basic")
}

func TestBasicAuthOption(t *testing.T) {
	client, err := New(TestBaseURL, BasicAuth("username", "password"))
	assert.NoError(t, err)
	assert.Equal(t, "username", client.username, "Username not configured")
	assert.Equal(t, "password", client.password, "Password not configured")

	request, err := http.NewRequest("GET", "https://test.com", nil)
	assert.NoError(t, err)
	client.configureAuth(request)

	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("username:password"))
	assert.Equal(t, auth, request.Header.Get("Authorization"), "Basic Auth not properly configured")
}

func TestGetConfig(t *testing.T) {
	mockHTTPClientFactory()
	defer restoreDefaultHTTPClientFactory()

	httpmock.RegisterResponder(
		"GET", TestBaseURL+"label/app-p1,p2.json", httpmock.NewStringResponder(
			200, `{"services": "myapp", "key": "value"}`))

	client, _ := New(TestBaseURL)
	config, _ := client.GetConfig("app", "label", "p1", "p2")
	assert.Equal(t, `{"services": "myapp", "key": "value"}`, string(config))

	config, _ = client.GetConfig("app", "label")
	assert.Equal(t, `{"key": "value"}`, string(config))
}

func TestGetConfigFile(t *testing.T) {
	mockHTTPClientFactory()
	defer restoreDefaultHTTPClientFactory()

	httpmock.RegisterResponder(
		"GET", TestBaseURL+"app/p1,p2/label/file.txt", httpmock.NewStringResponder(
			200, `SOME_FILE_CONTENT`))

	client, _ := New(TestBaseURL)
	config, _ := client.GetConfigFile("file.txt", "app", "label", "p1", "p2")
	assert.Equal(t, `SOME_FILE_CONTENT`, string(config))
}

func TestGetApps(t *testing.T) {
	mockHTTPClientFactory()
	defer restoreDefaultHTTPClientFactory()

	client, _ := New(TestBaseURL)
	// happy path - list with two apps
	httpmock.RegisterResponder(
		"GET", TestBaseURL+"label/app-p1,p2.json", httpmock.NewStringResponder(
			200, `{"services": [ "app-1", "app-2" ], "key": "value"}`))
	apps, err := client.getApps("services", "app", "label", "p1", "p2")
	assert.NoError(t, err)
	assert.Equal(t, []string{"app-1", "app-2"}, apps)

	// string with a single app
	httpmock.RegisterResponder(
		"GET", TestBaseURL+"label/app-p1,p2.json", httpmock.NewStringResponder(
			200, `{"services": "app-1", "key": "value"}`))
	apps, err = client.getApps("services", "app", "label", "p1", "p2")
	assert.NoError(t, err)
	assert.Equal(t, []string{"app-1"}, apps)

	// object instead of string
	httpmock.RegisterResponder(
		"GET", TestBaseURL+"label/app-p1,p2.json", httpmock.NewStringResponder(
			200, `{"services": { "key": "value" }, "key": "value"}`))
	apps, err = client.getApps("services", "app", "label", "p1", "p2")
	assert.Error(t, err)
	assert.Nil(t, apps)

	// no app list
	httpmock.RegisterResponder(
		"GET", TestBaseURL+"label/app-p1,p2.json", httpmock.NewStringResponder(200, `{"key": "value"}`))
	apps, err = client.getApps("services", "app", "label", "p1", "p2")
	assert.Error(t, err)
	assert.Nil(t, apps)

	// no yaml
	httpmock.RegisterResponder(
		"GET", TestBaseURL+"label/app-p1,p2.json", httpmock.NewStringResponder(200, `SOME TEXT`))
	apps, err = client.getApps("services", "app", "label", "p1", "p2")
	assert.Error(t, err)
	assert.Nil(t, apps)

	// 403 Unauthorized
	httpmock.RegisterResponder(
		"GET", TestBaseURL+"label/app-p1,p2.json", httpmock.NewStringResponder(403, ""))
	apps, err = client.getApps("services", "app", "label", "p1", "p2")
	assert.Error(t, err)
	assert.Nil(t, apps)

}

func TestInterfaceToStringSlice(t *testing.T) {
	i := []interface{}{"apa", "banan"}
	s := interfaceToStringSlice(i)
	assert.Equal(t, []string{"apa", "banan"}, s)
}

func TestAppendYAMLDoc(t *testing.T) {
	config := appendYAMLDoc("app", []byte(""), []byte("Some fake YAML\n"))
	assert.Equal(t, "---\nSome fake YAML\n", string(config))

	config = appendYAMLDoc("app", config, []byte("More fake YAML"))
	assert.Equal(t, "---\nSome fake YAML\n---\nMore fake YAML\n", string(config))

	config = appendYAMLDoc("app", config, []byte("---\nLeading doc separator"))
	assert.Equal(t, "---\nSome fake YAML\n---\nMore fake YAML\n---\nLeading doc separator\n", string(config))

	config = appendYAMLDoc("app", []byte(""), []byte("\n---\nLeading newline with doc separator\n"))
	assert.Equal(t, "---\nLeading newline with doc separator\n", string(config))

	config = appendYAMLDoc("app", []byte("---\nSome fake YAML\n"), []byte(""))
	assert.Equal(t, "---\nSome fake YAML\n", string(config), "Empty document appended")

	config = appendYAMLDoc("app", []byte("---\nSome fake YAML\n"), []byte("---\n"))
	assert.Equal(t, "---\nSome fake YAML\n", string(config), "Empty document with separator appended")
}

// -- suport

func restoreDefaultHTTPClientFactory() {
	httpClientFactory = defaultHTTPClientFactory
	httpmock.Reset()
	httpmock.DeactivateAndReset()
}

func mockHTTPClientFactory() {
	httpClientFactory = func() *http.Client {
		return http.DefaultClient
	}
	httpmock.Activate()
	httpmock.RegisterNoResponder(httpmock.NewStringResponder(200, `{"key": "value"}`))
}
