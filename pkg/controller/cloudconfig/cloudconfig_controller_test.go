package cloudconfig

import (
    "testing"
	"github.com/stretchr/testify/assert"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFallBackIfEmpty(t *testing.T) {
	val := struct{ field string }{""}
	fallBackIfEmpty(&val.field, "aValue")
	assert.Equal(t, "aValue", val.field)
}

func TestSetFallbackValues(t *testing.T) {
	config := k8v1alpha1.CloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dms",
		},
	}
	setFallbackValues(&config)
	assert.Equal(t, "dms", config.Spec.Name)
	assert.Equal(t, "dms", config.Spec.AppName)
	assert.Equal(t, "dms", config.Spec.Key)
    assert.Equal(t, DefaultSchedule, config.Spec.Schedule)

    // test that explicitely set values are not overwritten
    config.Spec.Environment.Name = "Microservices"
    config.Spec.Environment.AppName = "DMS"
    config.Spec.Environment.Key = "dms-cluster"
    config.Spec.Schedule = "1 0 0 0 0"

    setFallbackValues(&config)

	assert.Equal(t, "Microservices", config.Spec.Name)
	assert.Equal(t, "DMS", config.Spec.AppName)
	assert.Equal(t, "dms-cluster", config.Spec.Key)
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
