package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	TestBaseURL = "http://test.com/"
	TestEnv     = `
    server:   test.com
    profile:  [ p1, p2 ]
    label:    label
    specFile: deployment.yaml
    appName:  dms-cluster
    appList:  services
  `
	TestSpec = `
    server: cloud-config-server # Cloud Config Server name or URL
    credentials: cloud-config   # Cloud Config Server secret
    label: master               # label used for all apps, defaults to 'master'
    specFile: deployment.yaml   # app spec file, defaults to 'deployment.yaml'
    appName: dms-cluster        # application name, defaults to the CloudConfig name
    appList: services           # application list property of AppName app

    environments:                 # Environments where apps are managed, global values can be overridden
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
)

// -- helper methods

// k8MarshalYAML marshals YAML using the K8 YAML and JSON utility functions
func k8MarshalYAML(t *testing.T, spec string, obj interface{}) {
	b := []byte(spec)
	j, err := yaml.ToJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(j, obj)
}
