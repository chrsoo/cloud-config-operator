# Cloud Config Operator
Provision Kubernetes applications using Spring Cloud Config Server 
with a GitOps approach.

__This is currently just a skeleton project!__

 
## Custom Resource Definitions
### CloudConfig
The CloudConfig CRD defines a Cloud Config Server configuration that Spring Cloud
Operator will monitor and synchronize with one or more Kubernetes applications in
a number of `environments`.

An __environment__ is the Cloud Config configuration given by a `profile` and 
`label`. For each environment in `environments`, Spring Cloud Operator will

* Retrieve the `appList` list of applications in the `appName` applicaiton configuration; and
* For each application on in the apply the `specFile` Kubernetes specification

CloudConfig CRD example:

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
### CloudConfigServer
The CloudConfigServer CRD is used to manage a Spring Cloud Config Server Kubernetes service.

CloudConfigServer CRD example:
```yaml
apiVersion: k8.jabberwocky.se/v1alpha1
kind: CloudConfigServer
metadata:
  name: cloud-config-server   # name of this config server
spec:
  # TODO provide sample configuration of a Spring Cloud Server
```

## References
* [Quick Intro to Spring Cloud Config](https://www.baeldung.com/spring-cloud-configuration)
