package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudConfigSpec(t *testing.T) {
	var actual CloudConfigSpec
	k8MarshalYAML(t, TestSpec, &actual)

	assert.Equal(t, "dms-cluster", actual.Environment.AppName,
		"Incorrect default value")

	assert.Equal(t, "dms-cluster", actual.GetEnvironment("dev").AppName,
		"Unspecified value for AppName should fall back to global default")
}

func TestEnvironment(t *testing.T) {
	var actual Environment
	k8MarshalYAML(t, TestEnv, &actual)
	assert.Equal(t, "dms-cluster", actual.AppName)
}
