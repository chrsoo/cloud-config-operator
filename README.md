# Cloud Config Operator
Provision Kubernetes applications using Spring Cloud Config Server with a GitOps approach.

## Custom Resource Definition
The `CloudConfig` CRD defines one or more Spring Cloud Config Apps that that
Cloud Config Operator will monitor and synchronize with the cluster state.

CloudConfig CRD example:

```yaml
apiVersion: k8.jabberwocky.se/v1alpha1
kind: CloudConfig
metadata:
  name: test
spec:
  server: cloud-config-server:8888  # Cloud Config Server name or URL
  secert: cloud-config              # Cloud Config Server secret
  label: master                     # label used for all apps, defaults to 'master'
  specFile: deployment.yaml         # app spec file, defaults to 'deployment.yaml'
  appName: cluster                  # application name, defaults to the CloudConfig name
  appList: services                 # application list property of AppName app
  insecure: true                    # do not require or verify SSL server certificates

  environments:                     # Environments where apps are managed, global values can be overridden
    dev:                            # Map key is used as the environment's name
      profile: [ dev ]              # cloud config profiles for the env
    prd:
      profile: [ prd ]
```
## Usage

Cloud Config Apps exist in a number of `environments`. Each environment is
defined by a list of `profiles` and a `label`.

For each environment in `environments`, Spring Cloud Operator will

* Retrieve the app's `specFile` from the Spring Cloud Config Server
* Apply the `specFile` to the Kubernetes cluster

### App or List of Apps

A CloudConfig can either define a single App or a list of Apps.

In the first case the `appName` is the name of the Spring Cloud Config application

In the second case the `appName` defines the application that contains a single
configuration property `appList` that lists the Apps to manage. This property
can have different values in different environments.

### Environments and namespaces
TODO

### Versioning per environment
TODO

### Different spec files per environment
TODO

## REST API

(!) Planned for v0.4!

### Refresh config for all apps in all environments

    POST /config/{name}

Optionally a JSON message in the body of the HTTP request can be used to restrict what environments and apps are refreshed.

```json
{
  "label": "develop",
  "env": [ "dev" ],
  "app": [ "alpha", "beta" ]
}
```

* `label` - only environments with a matching label are affected.
* `env` - only environments provided in the array are affected
* `app` - only applications provided in the array are affected

### Refresh config for all apps in a given environment

    POST /config/{name}/env/{env}

### Refresh config for all apps and all environments using a given label

    POST /config/{name}/label/{label}

### Refresh config for a single app in a given environment

    POST /config/{name}/env/{env}/app/{app}

### Refresh config for a single app in all environments

    POST /config/{name}/app/{app}

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

- [X] v0.1 `CloudConfig` CRD for managing Spring Cloud Config apps. Experimental version based on CronJob and Kubectl to get things going.
- [ ] v0.2 `CloudConfigEnv` CRD replaces CronJob.
- [ ] v0.3 `CloudConfigApp` CRD removes the need for `kubectl` command
- [ ] v0.4 REST API for CI/CD integrations
- [ ] v1.0 The first stable version of `cloud-config-operator` resource API

## References
* [operator-sdk](https://github.com/operator-framework/operator-sdk)
* [Quick Intro to Spring Cloud Config](https://www.baeldung.com/spring-cloud-configuration)
* [Best practices for building Kubernetes Operators and stateful apps](https://cloud.google.com/blog/products/containers-kubernetes/best-practices-for-building-kubernetes-operators-and-stateful-apps)
