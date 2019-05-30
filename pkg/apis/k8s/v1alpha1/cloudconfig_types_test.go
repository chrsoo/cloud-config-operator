package v1alpha1

import (
	"testing"
)

// -- test harness

func getTestEnv(t *testing.T) CloudConfig {
	var env CloudConfig
	k8MarshalYAML(t, TestEnv, &env)
	return env
}
