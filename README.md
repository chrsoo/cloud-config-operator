# Cloud Config Operator
Provision Kubernetes applications using Spring Cloud Config Server
with a GitOps approach.

## Custom Resource Definitions
### CloudConfig
The CloudConfig CRD defines a Cloud Config Server configuration that Spring Cloud
Operator will monitor and synchronize with the cluster state. A CloudConfig
specifies a number of `environments`. Each environment is in turn defined by a
list of `profiles` and a `label`.

For each environment in `environments`, Spring Cloud Operator will

* Retrieve the `appList` list of applications for the `appName` application; and
* Apply the `specFile` Kubernetes specification for each app on the list

CloudConfig CRD example:

```yaml
apiVersion: k8.jabberwocky.se/v1alpha1
kind: CloudConfig
metadata:
  name: dms                     # System or Application name
spec:
  server: cloud-config-server   # Cloud Config Server name or URL
  secret: cloud-config          # Cloud Config Server secret
  label: master                 # label used for all apps, defaults to 'master'
  specFile: deployment.yaml     # app spec file, defaults to 'deployment.yaml'
  appName: dms-cluster          # main app name, defaults to the CloudConfig name
  appList: services             # app list property
  schedule: "*/1 * * * *"       # cron job schedule
  environments:                 # app environments, global values can be overridden
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
```
## Project Setup
The following instructions assume Mac OS X with [Home Brew](https://brew.sh/) and a local [Minikube](https://github.com/kubernetes/minikube) as the development Kubernetes cluster:

    brew install cask minikube
    brew install kubernetes-cli
    brew install docker
    brew install golang dep

Once you have a working Go environment you can get the project using

    go get github.com/chrsoo/cloud-config-operator

__To setup the Operator SDK please follow the [Quick Start instsructions on GitHub](https://github.com/operator-framework/operator-sdk#quick-start)!__

## Roadmap
Planned release versions:

- [ ] v0.1 `CloudConfig` CRD for managing Spring Cloud Config apps
- [ ] v1.0 The first stable version of `cloud-config-operator` resource API

## References
* [operator-sdk](https://github.com/operator-framework/operator-sdk)
* [Quick Intro to Spring Cloud Config](https://www.baeldung.com/spring-cloud-configuration)
* [Best practices for building Kubernetes Operators and stateful apps](https://cloud.google.com/blog/products/containers-kubernetes/best-practices-for-building-kubernetes-operators-and-stateful-apps)
