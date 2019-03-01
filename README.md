# Cloud Config Operator
Provision Kubernetes applications using Spring Cloud Config Server with a GitOps approach.

## Custom Resource Definition
The `CloudConfig` CRD defines one or more Spring Cloud Config Apps that that
Cloud Config Operator will monitor and synchronize with the cluster state.

CloudConfig CRD example:

```yaml
apiVersion:     k8s.jabberwocky.se/v1alpha1
kind:           CloudConfig
metadata:
  name:         test                      # CloudConfig name
spec:
  server:       cloud-config-server:8888  # Cloud Config Server name or URL
  credentials:                            # Cloud Config Credentials
    secret:     cloud-config-secret       # Name of the credential secret, required for credentials
    token:      token                     # Name of the token entry, defaults to `token`
    username:   username                  # Name of the username entry, defaults to `username`
    password:   password                  # Name of the password entry, defaults to `password`
    cert:       cert.pem                  # Name of the cert entry, defaults to `cert.pem`
    key:        cert.key                  # Name of the private key entry, defaults to `cert.key`
    rootCA:     ca.pem                    # Name of the CA cert entry, defaults to `ca.pem`
  appName:      cluster                   # app name, defaults to the CloudConfig name
  label:        master                    # label used for all apps, defaults to 'master'
  appList:      services                  # app list property in the app config
  specFile:     deployment.yaml           # app spec file, defaults to 'deployment.yaml'
  insecure:     true                      # do not require or verify SSL server certs
  truststore:   global-trust-store        # Optional secret containg all trusted certs
  period:       10                        # seconds between configuation cycles, defaults to 0 (disabled)

  environments:                           # Environments where apps are managed, defaults to global conf
    dev:                                  # environment key
      name:     Development               # environment name, defaults to the key value
      profile:  [ dev ]                   # cloud config profiles for the env
    prd:
      profile:  [ prd ]
```
## Overview
Cloud Config Apps exist in a number of `environments`. Each environment is
defined by a list of `profiles` and a `label`.

For each *environment* in `environments`, Spring Cloud Operator will

* Retrieve the `appList` list of applications for the `appName` application;
* Retrieve the `specFile` Kubernetes YAML file for each app on the list;
* Pipe the concatenated YAML spec files to the following `kubectl` command:
  ```
  kubectl apply -ns <environment> --purge -f -
  ```
(Ideally we should get rid of using `kubectl` and instead implement the synchronization logic in the operator using the Kubernetes API machinery; this is subject to a future release)
### Spring Cloud Config Example

The examples that follow assume a Spring Cloud Config Server backed by a Git repository (or file system) similar to the [test repository](test/server/repository). This file repository can be used to back a Spring Cloud Config Server test deployment as found in the [cloud-config-server.yaml](test/deploy/cloud-config-server.yaml) file.

:warning:   Note the Spring Cloud Config Server deployment is intended for testing and that correctly implementing Spring Cloud Config Server is out of scope for this project!

### App or List of Apps

A CloudConfig can either define a single App or a list of Apps.

In the first case the `appName` is the name of the Spring Cloud Config application.

In the second case the `appName` defines the application that must contain at least the configuration property defined by `appList` that lists the Apps to manage. This property
can have different values in different environments.

### Environments and Namespaces
An environment manages a single namespace named after the `CloudConfig` and the environment. For example given the example above with the `CloudConfig` name `test` the `prd` environment manages the namespace  `test-prd`

### Versioning of Apps
As we typically want to propagate a new App version through the different environments, e.g. from `dev` to `prd` (in many enterprises there would also be a `qua` for quality assurance etc) it makes sense to make the App version configurable per environment.

In the configuration repository this could be managed by having the file `alpha-dev.yaml` contain the property field
```
version: 1.2.0-SNAPSHOT
```
... and `alpha-prd.yaml` contain the current stable version
```
version: 1.1.0
```
### Kubernetes specification files
In order to manage Kubernetes deployments for the apps we use the `specFile` property. This contains the name of the Spring Cloud Config template file used to deploy the apps, e.g. [deployment.yaml](test/server/repository/deployment.yaml).

The template file contains property placeholders which will be filed in by the configuration values for which the file is retrieved.

Note that a deployment file can be replaced for each application (cf. the [deployment.yaml](test/server/repository/gamma/deployment.yaml) for the gamma example app.) The file can also be different per environment etc.

## Install

```
# Add the CRD's
kubectl apply -f deploy/crds/k8_v1alpha1_cloudconfig_crd.yaml -f deploy/crds/k8_v1alpha1_cloudconfigenv_crd.yaml

# Don't forget to review and adapt role.yaml before applying to the cluster!
kubectl apply -f deploy/role.yaml -f deploy/role_binding.yaml -f deploy/service_account

# Deploy the operator
sed 's|REPLACE_IMAGE|dtr.richemont.com/digital/cloud-config-operator:0.2.0|g' deploy/operator.yaml | kubectl apply -f -
```

###Adapt the cloud-config-operator role
The `cloud-config-operator` role defined in [deploy/role.yaml] should be adapted to the specific needs of your Kubernetes cluster. The following permissions are required for the basic operation of the `cloud-config-operator`

// TODO tune the rules required for the operation of cloud-config-operator
- watch, list , retrieve, create, update and delete namespaces
  ```yaml
  // TODO add role yaml rules
  ```
In addition the operator needs to have all the required permissions to manage the apps. Typically this means creating, retrieving and deleting deployments but additional rules  may be required if for example the apps define `CronJob`s in their YAML specifications.

## REST API

:warning: Planned for v0.3!

Posting to the URI

    POST /config/{name}

Will refresh all Apps for all Environments for the given CloudConfig `name`

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

Restrictions can also be applied by adding `label`, `env` and `app` to the URI path:

| Request | Usecase |
| ------- | ------- |
| `POST /config/{name}/env/{env}` | Refresh config for all apps in a given environment |
| `POST /config/{name}/label/{label}` | Refresh config for all apps and all environments using a given label |
| `POST /config/{name}/env/{env}/app/{app}` | Refresh config for a single app in a given environment |
| `POST /config/{name}/app/{app}` | Refresh config for a single app in all environments |

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
- [X] v0.2 `CloudConfigEnv` CRD replaces CronJob.
- [ ] v0.3 REST API for CI/CD integrations
- [ ] v0.4 `CloudConfigApp` CRD removes the `kubectl` dependency
- [ ] v1.0 The first stable version of `cloud-config-operator` resource API

## References
* [operator-sdk](https://github.com/operator-framework/operator-sdk)
* [Quick Intro to Spring Cloud Config](https://www.baeldung.com/spring-cloud-configuration)
* [Best practices for building Kubernetes Operators and stateful apps](https://cloud.google.com/blog/products/containers-kubernetes/best-practices-for-building-kubernetes-operators-and-stateful-apps)
