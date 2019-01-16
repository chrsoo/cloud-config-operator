package v1alpha1

import (
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestReconcile(t *testing.T) {
	execCommand = fakeExecCommand
	defer func() { execCommand = exec.Command }()

	activateAndMockConfigServerResponses(t)
	defer httpmock.DeactivateAndReset()

	env := getTestEnv(t)
	assert.Equal(t, []string{"article", "authz"}, env.getApps())

	go env.reconcile()
}

func TestAppendYAMLDoc(t *testing.T) {
	config := appendYAMLDoc([]byte(""), []byte("Some fake YAML\n"))
	assert.Equal(t, "---\nSome fake YAML\n", string(config))

	config = appendYAMLDoc(config, []byte("More fake YAML"))
	assert.Equal(t, "---\nSome fake YAML\n---\nMore fake YAML\n", string(config))

	config = appendYAMLDoc(config, []byte("---\nLeading doc separator"))
	assert.Equal(t, "---\nSome fake YAML\n---\nMore fake YAML\n---\nLeading doc separator\n", string(config))

	config = appendYAMLDoc([]byte(""), []byte("\n---\nLeading newline with doc separator\n"))
	assert.Equal(t, "---\nLeading newline with doc separator\n", string(config))

	config = appendYAMLDoc([]byte("---\nSome fake YAML\n"), []byte(""))
	assert.Equal(t, "---\nSome fake YAML\n", string(config), "Empty document appended")

	config = appendYAMLDoc([]byte("---\nSome fake YAML\n"), []byte("---\n"))
	assert.Equal(t, "---\nSome fake YAML\n", string(config), "Empty document with separator appended")
}

// TODO move to cloudconfg_test.go
func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TODO move to cloudconfg_test.go
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// TODO capture input from stdin
	if os.Args[4] == "get" {
		os.Exit(1)
	}
	os.Exit(0)
}

// TODO test error codes, in particualar 401, 403 and 404
func TestGetApps(t *testing.T) {
	activateAndMockConfigServerResponses(t)
	defer httpmock.DeactivateAndReset()

	env := getTestEnv(t)
	assert.Equal(t, []string{"article", "authz"}, env.getApps())

	httpmock.RegisterResponder("GET", TestBaseURL+"label/cluster-p1,p2.yaml",
		httpmock.NewStringResponder(200, `{"services": "myapp", "key": "value"}`))
	assert.Equal(t, []string{"myapp"}, env.getApps())

	httpmock.RegisterResponder("GET", TestBaseURL+"label/cluster-p1,p2.yaml",
		httpmock.NewStringResponder(200, `{"services": { "nested": "value"}, "key": "value"}`))
	assert.Equal(t, []string{}, env.getApps())

	httpmock.RegisterResponder("GET", TestBaseURL+"label/cluster-p1,p2.yaml",
		httpmock.NewStringResponder(200, `[{"services": { "nested": "value"}, "key": "value"}]`))
	assert.Equal(t, []string{}, env.getApps(), "Does not handle invalid config")

	httpmock.RegisterResponder("GET", TestBaseURL+"label/cluster-p1,p2.yaml",
		httpmock.NewStringResponder(200, ``))
	assert.Equal(t, []string{}, env.getApps(), "Does not handle empty config")

	env.AppName = "appNotConfigured"
	assert.Equal(t, []string{}, env.getApps())
}

func TestGetAppConfig(t *testing.T) {
	activateAndMockConfigServerResponses(t)
	defer httpmock.DeactivateAndReset()

	env := getTestEnv(t)

	assert.Equal(t, `{"services": ["article", "authz"], "key": "value"}`, string(env.getAppConfig("cluster")),
		"JSON config not returned for environment")

	assert.Equal(t, `{"key": "value"}`, string(env.getAppConfig("anotherApp")))
}

func TestGetAppConfigFile(t *testing.T) {
	activateAndMockConfigServerResponses(t)
	defer httpmock.DeactivateAndReset()

	env := getTestEnv(t)

	assert.Equal(t, `YAML YAML YAML`, string(env.getAppConfigFile("cluster", env.SpecFile)),
		"YAML config file not returned from enironment")
}

func TestEnvBaseURL(t *testing.T) {
	env := getTestEnv(t)
	assert.Equal(t, TestBaseURL, env.baseURL())
	env.Server = "https://test.com/"

	assert.Equal(t, "https://test.com/", env.baseURL())
}

func TestInterfaceToStringSlice(t *testing.T) {
	i := []interface{}{"apa", "banan"}
	s := interfaceToStringSlice(i)
	assert.Equal(t, []string{"apa", "banan"}, s)
}

func TestConfigureAuth(t *testing.T) {
	request, _ := http.NewRequest("GET", "http://test.com", nil)
	env := Environment{Secret: "bogus-path"}
	assert.Panics(t, func() { env.configureAuth(request) }, "Expected panic when the path does not exist")

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-config-test-")
	assert.Nil(t, err, "Could not create temporary credentials dir")
	defer os.RemoveAll(tmpDir)

	env = Environment{Secret: tmpDir}
	assert.Panics(t, func() { env.configureAuth(request) }, "Expected panic when the username secret does not exist")

	err = ioutil.WriteFile(tmpDir+"/username", []byte("anonymous"), os.ModePerm)
	assert.Nil(t, err, "Could not write username secret")
	assert.Panics(t, func() { env.configureAuth(request) }, "Expected panic when the pasword secret does not exist")

	err = ioutil.WriteFile(tmpDir+"/password", []byte("secret"), os.ModePerm)
	assert.Nil(t, err, "Could not write password secret")
	env.configureAuth(request)
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("anonymous:secret"))
	assert.Equal(t, auth, request.Header.Get("Authorization"),
		"Basic auth should be configured if username and password secrets exist")

	err = ioutil.WriteFile(tmpDir+"/token", []byte("TOKEN"), os.ModePerm)
	assert.Nil(t, err, "Could not write token secret")
	env.configureAuth(request)
	assert.Equal(t, "Bearer TOKEN", request.Header.Get("Authorization"), "A token secret should override basic auth ")
}

func TestGetSecretPath(t *testing.T) {
	env := getTestEnv(t)
	env.Secret = "secret"
	assert.Equal(t, SecretPathPrefix+env.Secret, env.getSecretPath())

	env.Secret = "/some/arbitrary/path"
	assert.Equal(t, "/some/arbitrary/path", env.getSecretPath(), "The secret path should be the secret if it contains a slash")

	env.Secret = "/some/arbitrary/path/"
	assert.Equal(t, "/some/arbitrary/path", env.getSecretPath(), "Trailing slashes should be trimmed off the secret path")

}

func TestConfigure(t *testing.T) {
	env := Environment{}
	env.Configure()

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-config-test-")
	assert.Nil(t, err, "Could not create temporary credentials dir")
	defer os.RemoveAll(tmpDir)

	env = Environment{Insecure: true}
	env.Configure()
	env = Environment{Secret: tmpDir}
	env.Configure()
	env.Insecure = true
	env.Configure()
	// assert.Panics(t, func() { env.Configure() }, "Expected panic when the username secret does not exist")
}

// -- test harness

func activateAndMockConfigServerResponses(t *testing.T) {
	httpmock.Activate()

	httpmock.RegisterResponder("GET", TestBaseURL+"label/cluster-p1,p2.yaml",
		httpmock.NewStringResponder(200, `{"services": ["article", "authz"], "key": "value"}`))

	httpmock.RegisterResponder("GET", TestBaseURL+"cluster/p1,p2/label/deployment.yaml",
		httpmock.NewStringResponder(200, `YAML YAML YAML`))

	httpmock.RegisterResponder("GET", TestBaseURL+"authz/p1,p2/label/deployment.yaml",
		httpmock.NewStringResponder(200, `Authz YAML`))

	httpmock.RegisterResponder("GET", TestBaseURL+"article/p1,p2/label/deployment.yaml",
		httpmock.NewStringResponder(200, `Article YAML`))

	httpmock.RegisterNoResponder(
		httpmock.NewStringResponder(200, `{"key": "value"}`))
}

func getTestEnv(t *testing.T) Environment {
	var env Environment
	k8MarshalYAML(t, TestEnv, &env)
	return env
}
