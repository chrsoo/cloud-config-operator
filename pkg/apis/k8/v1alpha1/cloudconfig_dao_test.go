package v1alpha1

import (
	// yaml "gopkg.in/yaml.v2"
	// "encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

// TODO test error codes, in particualar 401, 403 and 404

func TestEnvironmentAppConfig(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	var env Environment
	k8MarshalYAML(t, TestEnv, &env)

	httpmock.RegisterResponder("GET", TestBaseURL+"app/p1,p2/label",
		httpmock.NewStringResponder(200, `{"services": ["article", "authz"]}`))

	httpmock.RegisterResponder("GET", TestBaseURL+"app/p1,p2/label/deployment.yaml",
		httpmock.NewStringResponder(200, `YAML YAML YAML`))

	assert.Equal(t, `{"services": ["article", "authz"]}`, string(env.GetAppConfig("app")),
		"JSON config not returned for environment")

	assert.Equal(t, `YAML YAML YAML`, string(env.GetAppConfigFile("app", env.SpecFile)),
		"YAML config file not returned from enironment")

	assert.Equal(t, []byte{}, env.GetAppConfig("bogus"))
}
func TestEnvBaseURL(t *testing.T) {
	var env Environment
	k8MarshalYAML(t, TestEnv, &env)

	assert.Equal(t, TestBaseURL, env.BaseURL())
	env.Server = "https://test.com/"

	assert.Equal(t, "https://test.com/", env.BaseURL())
}
