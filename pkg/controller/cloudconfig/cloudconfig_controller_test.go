package cloudconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8s/v1alpha1"
)

func TestValidation(t *testing.T) {
	spec := k8v1alpha1.CloudConfigSpec{}
	assert.Error(t, validate(&spec), "A Config Server URL is required")

	spec.Server = "http://localhost"
	assert.Error(t, validate(&spec), "The Config Server must use the https protocol if insecure=false")

	spec.Insecure = true
	assert.NoError(t, validate(&spec), "Insecure Config Server URLs are allowed is insecure=true")

	spec.Server = "https://localhost"
	assert.NoError(t, validate(&spec), "A secure Config Server URL should not provoke an error")
	spec.Insecure = false
	assert.NoError(t, validate(&spec), "A secure Config Server URL should not provoke an error")
}
