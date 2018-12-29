package v1alpha1

import (
	"encoding/json"
	"testing"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const env = `
  server: cloud-config-server # Cloud Config Server name or URL
  credentials: cloud-config   # Cloud Config Server secret 
  label: master               # label used for all apps, defaults to 'master'
  specFile: deployment.yaml   # app spec file, defaults to 'deployment.yaml'
  appName: dms-cluster        # application name, defaults to the CloudConfig name
  appList: services           # application list property
  `
const spec = `
    defaults:
      server: cloud-config-server # Cloud Config Server name or URL
      credentials: cloud-config   # Cloud Config Server secret 
      label: master               # label used for all apps, defaults to 'master'
      specFile: deployment.yaml   # app spec file, defaults to 'deployment.yaml'
      appName: dms-cluster        # application name, defaults to the CloudConfig name
      appList: services           # application list property
    
      environments:               # Environments where apps are managed, global values can be overridden
      dev:                        # environment key
        name: Development         # environment name, defaults to the key value
        profile: [ vsg, dev ]     # cloud config profiles for the env
        label: develop            # optionally override the global label
      qua:
        name: Quality
        profile: [ vsg, qua ]
      val:
        name: Validation
        profile: [ vsg, val ]
      prd:
        name: Production
        profile: [ vsg, prd ]
    `

func TestCloudConfigSpec(t *testing.T) {
  b := []byte(spec)
  j, err := yaml.ToJSON(b)
	if err != nil {
    t.Fatal(err)
	}
  var s CloudConfigSpec
	json.Unmarshal(j, &s)
	assert.Equal(t, "dms-cluster", s.Defaults.AppName)
}

func TestCloudConfigEnv(t *testing.T) {
  b := []byte(env)
  j, err := yaml.ToJSON(b)
	if err != nil {
    t.Fatal(err)
	}
  var e CloudConfigEnv
	json.Unmarshal(j, &e)
	assert.Equal(t, "dms-cluster", e.AppName)
}
