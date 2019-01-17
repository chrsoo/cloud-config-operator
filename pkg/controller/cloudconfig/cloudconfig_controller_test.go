package cloudconfig

import (
	"github.com/stretchr/testify/assert"
	"testing"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFallBackIfEmpty(t *testing.T) {
	val := struct{ field string }{""}
	fallBackIfEmpty(&val.field, "aValue")
	assert.Equal(t, "aValue", val.field)
}

func TestSetDefaults(t *testing.T) {
	config := k8v1alpha1.CloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dms",
		},
	}
	setDefaults(&config)
	assert.Equal(t, config.ObjectMeta.Name, config.Spec.Name, "Spec name should always be the config's metadata name")
	assert.Equal(t, "dms", config.Spec.AppName)
	assert.Equal(t, DefaultSchedule, config.Spec.Schedule)

	// test that explicitely set values are not overwritten
	config.Spec.Environment.Name = "Microservices"
	config.Spec.Environment.AppName = "DMS"
	config.Spec.Schedule = "1 0 0 0 0"

	setDefaults(&config)

	assert.Equal(t, config.ObjectMeta.Name, config.Spec.Name, "Spec name should always be the config's metadata name")
	assert.Equal(t, "DMS", config.Spec.AppName)
	assert.Equal(t, "1 0 0 0 0", config.Spec.Schedule)

}

func TestNewCronJobForCR(t *testing.T) {
	config := k8v1alpha1.CloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dms",
		},
	}
	job, err := newCronJobForCR(&config)
	assert.Nil(t, err, "Err should be nil")
	assert.NotNil(t, job, "Job should be an instance")
	assert.Equal(t, config.Spec.Schedule, job.Spec.Schedule)
}

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
