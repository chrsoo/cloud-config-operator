package v1alpha1

import (
	"os/exec"
	"testing"
	httpmock "gopkg.in/jarcoal/httpmock.v1"

	"github.com/stretchr/testify/assert"
)

func TestCloudConfigSpec(t *testing.T) {
	var actual CloudConfigSpec
	k8MarshalYAML(t, TestSpec, &actual)

	assert.Equal(t, "dms-cluster", actual.AppName,
		"Incorrect default value")

	dev := actual.GetEnvironment("dev")
	assert.Equal(t, "dms-cluster", dev.AppName, "Unspecified value for AppName should fall back to global default")
	assert.Equal(t, "Development", dev.Name, "Name value should not be overwritten")
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
	actual.Credentials = ""
	// mock kubectl command calls
	execCommand = fakeExecCommand
	defer func() { execCommand = exec.Command }()

	actual.Reconcile()
}
