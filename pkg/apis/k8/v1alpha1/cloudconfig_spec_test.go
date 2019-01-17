package v1alpha1

import (
	"os"
	"io/ioutil"
	"os/exec"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"

	"github.com/stretchr/testify/assert"
)

func TestCloudConfigSpec(t *testing.T) {
	var actual CloudConfigSpec
	k8MarshalYAML(t, TestSpec, &actual)

	assert.Equal(t, "cluster", actual.AppName, "Incorrect default value")

	dev := actual.GetEnvironment("dev")
	assert.Equal(t, "cluster", dev.AppName, "Unspecified value for AppName should fall back to global default")
	assert.Equal(t, "dev", dev.Name, "Name value should same as the environments key")
	assert.Equal(t, "develop", dev.Label, "Label value should not be overwritten")

	qua := actual.GetEnvironment("qua")
	assert.Equal(t, "qua", qua.Name, "Name value should fall back to env Key")
	assert.Equal(t, "master", qua.Label, "Label value should fall back to global default")
}

func TestReconcileSpec(t *testing.T) {
	var actual CloudConfigSpec
	k8MarshalYAML(t, TestSpec, &actual)

	// mock config server
	activateAndMockConfigServerResponses(t)
	defer httpmock.DeactivateAndReset()
	// avoid creating credentials on the file system!
	actual.Secret = ""
	// mock kubectl command calls
	execCommand = fakeExecCommand
	defer func() { execCommand = exec.Command }()

	actual.Reconcile()
}

func TestInit(t *testing.T) {
	spec := CloudConfigSpec{}
	spec.Init()

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-config-test-")
	assert.Nil(t, err, "Could not create temporary credentials dir")
	defer os.RemoveAll(tmpDir)

	spec = CloudConfigSpec{Environment: Environment{Insecure: true}}
	spec.Init()
	spec = CloudConfigSpec{Environment: Environment{Secret: tmpDir}}
	spec.Init()
	spec.Insecure = true
	spec.Init()
	// assert.Panics(t, func() { spec.Init() }, "Expected panic when the username secret does not exist")
}
