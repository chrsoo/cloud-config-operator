# Cloud Config Operator
Provision Kubernetes applications using Spring Cloud Config Server 
with a GitOps approach.

__This is currently just a skeleton project!__

 
## Custom Resource Definitions
### CloudConfig
The CloudConfig CRD defines a Cloud Config Server configuration that Spring Cloud
Operator will monitor and synchronize with one or more Kubernetes applications in
a number of `environments`.

An environment is the configuration given by a `profile` and `label`.

For each environment, Spring Cloud Operator will

* Retrieve the `appList`list of applications for the `appName` application
* For each application apply the `specFile` Kubernetes specification

Cloud Config example:

```yaml
apiVersion: k8.jabberwocky.se/v1alpha1
kind: CloudConfig
metadata:
  name: dms                   # System or Application name 
spec:
  server: cloud-config-server # Cloud Config Server name or URL
  credentials: cloud-config   # Cloud Config Server secret 
  label: master               # label used for all apps, defaults to 'master'
  specFile: deployment.yaml   # app spec file, defaults to 'deployment.yaml'
  appName: dms-cluster        # application name, defaults to the CloudConfig name
  appList: services           # application list property
  environments:               # Environments where apps are managed
    dev:                      # environment key
      name: Development       # environment name, defaults to the key value
      profile: [ vsg, dev ]   # cloud config profiles for the env
      label: develop          # optionally override the global label
    qua:
      name: Quality
      profile: [ vsg, qua ]
    val:
      name: Validation
      profile: [ vsg, val ]
    prd:
      name: Production
      profile: [ vsg, prd ]
```

Given the example above and that `cloud-config-serfver` Spring Cloud Operator will 

1. Retrieve the file `/dms-cluster/vsg,dev/develop.yaml` configuration
1. and apply it to the `ns` namespace
1. Retrieve the configuration `/dms-cluster/vsg,dev/develop.yaml` and iterate over the `services` array
1. For each `app` in the array, apply the `${app}/vsg,dev/develop/deployment.yaml`

It then proceeds to the next namespace in the list

### CloudConfigServer
git
Example:

```yaml
apiVersion: k8.jabberwocky.se/v1alpha1
kind: CloudConfigServer
metadata:
  name: cloud-config-server   # name of this config server
spec:
   

```

## References
* [Quick Intro to Spring Cloud Config](https://www.baeldung.com/spring-cloud-configuration)
