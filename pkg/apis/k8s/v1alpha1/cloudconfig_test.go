package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	TestBaseURL = "https://test.com/"
	TestEnv     = `
    server:   test.com
    profile:  [ p1, p2 ]
    label:    label
    specFile: deployment.yaml
    appName:  cluster
    appList:  services
  `
	TestSpec = `
    server: cloud-config-server   # Cloud Config Server name or URL
    secret: cloud-config          # Cloud Config Server secret
    label: master                 # label used for all apps, defaults to 'master'
    profile: [ prd ]
    specFile: deployment.yaml     # app spec file, defaults to 'deployment.yaml'
    appName: cluster              # application name, defaults to the CloudConfig name
    appList: services             # application list property of AppName app
    `
	CloudConfigExample = `
  apiVersion: k8s.jabberwocky.se/v1alpha1
  kind: CloudConfig
  metadata:
    name: test
  spec:
    server: cloud-config-server:8888  # Cloud Config Server name or URL
    secret: cloud-config              # Cloud Config Server secret
    label: master                     # label used for all apps, defaults to 'master'
    profile: [ prd ]
    specFile: deployment.yaml         # app spec file, defaults to 'deployment.yaml'
    appName: cluster                  # application name, defaults to the CloudConfig name
    appList: services                 # application list property of AppName app
    insecure: true                    # do not require or verify SSL server certificates
    period: 10                        # time between reconciliation cycles  `
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
