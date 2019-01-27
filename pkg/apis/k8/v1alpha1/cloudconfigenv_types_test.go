package v1alpha1

import (
	"testing"
)

// -- test harness

func getTestEnv(t *testing.T) CloudConfigEnv {
	var env CloudConfigEnv
	k8MarshalYAML(t, TestEnv, &env)
	return env
}
