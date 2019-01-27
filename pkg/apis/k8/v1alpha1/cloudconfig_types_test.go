package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudConfig(t *testing.T) {
	var actual CloudConfig
	k8MarshalYAML(t, CloudConfigExample, &actual)
	assert.Equal(t, "cluster", actual.Spec.AppName)

	dev := actual.GetEnvironment("dev")
	assert.Equal(t, "cluster", dev.Spec.AppName, "Unspecified value for AppName should fall back to global default")
	assert.Equal(t, "test-dev", dev.Name, "Name value should be the confg name concatenated with the environment name")
	assert.Equal(t, "develop", dev.Spec.Label, "Label value should not be overwritten")
	assert.Equal(t, "cloud-config-server:8888", dev.Spec.Server, "Server should be inherited from parent")
	assert.Equal(t, 10, dev.Spec.Period, "Period should be inherited from parent")
	assert.True(t, dev.Spec.Insecure, "Period should be inherited from parent")

	qua := actual.GetEnvironment("prd")
	assert.Equal(t, "test-prd", qua.Name, "Name value should be the confg name concatenated with the environment name")
	assert.Equal(t, "master", qua.Spec.Label, "Label value should fall back to global default")

	nop := actual.GetEnvironment("nop")
	assert.Nil(t, nop)
}

func TestIsManagedNamespace(t *testing.T) {
	var actual CloudConfig
	k8MarshalYAML(t, CloudConfigExample, &actual)

	assert.True(t, actual.IsManagedNamespace("test-dev"))
	assert.False(t, actual.IsManagedNamespace("dev-test"))
	assert.False(t, actual.IsManagedNamespace(""))
}
