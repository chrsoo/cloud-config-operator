# Cloud Config Operator
Provision Kubernetes applications using Spring Cloud Config Server with a GitOps approach.

## Overview
The `CloudConfig` CR defines one or more Spring Cloud Config Applocations that
Cloud Config Operator will monitor and synchronize with the cluster state.

`CloudConfig` CR example:

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
  label:        master                    # cloud config label used for all apps, defaults to 'master'
  profile:      [ dev ]                   # cloud config application profile(s)
  appList:      services                  # app list property in the app config
  specFile:     deployment.yaml           # app spec file, defaults to 'deployment.yaml'
  insecure:     true                      # do not require or verify SSL server certs
  truststore:   global-trust-store        # Optional secret containg all trusted certs
  period:       10                        # seconds between configuation cycles, defaults to 0 (disabled)
```

Each time the CR is changed (or optionally every `period` number of seconds) the operator will

 * Retrieve the `specFile` Kubernetes YAML file for the given `appName` application, `label` and `profile` from the `server`;
 * Run the following `kubectl` command:
    ```
    kubectl apply -ns <namespace> --purge -f -
    ```

If the `CloudConfig` contains a Spring Cloud Config property in the `appList` field the operator will instead

* Retrieve the `appList` list of applications for the `appName` application, `label` and `profile` from the `server`;
* Retrieve and concatenate the `specFile` Kubernetes YAML file for each app on the list using the same `label` and `profile`;
* Pipe the concatenated YAML spec files to the following `kubectl` command:
  ```
  kubectl apply -ns <namespace> --purge -f -
  ```

## Spring Cloud Config Example

The examples that follow assume a Spring Cloud Config Server backed by a Git repository (or file system) similar to the [test repository](test/server/repository). This file repository can be used to back a Spring Cloud Config Server test deployment as found in the [cloud-config-server.yaml](test/deploy/cloud-config-server.yaml) file.

:warning: Note the Spring Cloud Config Server deployment is intended for testing and that correctly implementing Spring Cloud Config Server is out of scope for this project!

## Kubernetes specification files
In order to manage Kubernetes deployments for the apps we use the `specFile` property. This contains the name of the Spring Cloud Config template file used to deploy the apps, e.g. [deployment.yaml](test/server/repository/deployment.yaml).

The template file contains property placeholders which will be filed in by the configuration values for which the file is retrieved.

For example, the `app` label in this example snippet will be replaced by the value of the `${app}` placeholder:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${app}
  labels:
    app: ${app}
...
```

Note that a deployment file can be different for each application.

For example the gamma application in the example app is different than for the other apps, cf. the [deployment.yaml](test/server/repository/gamma/deployment.yaml) file.

## Application Versions
As we typically want to promote App versions through the different environments, e.g. from `dev` to `qua` to `prd`, it makes sense to make the Application version configurable per environment. Differente environments would match to different profiles.

For example given that the `alpha` application defines a  `version` property the corresponding configuration files in a Spring Cloud Config Git repository could look like...

__alpha-dev.yaml__

    version: 1.1.2-SNAPSHOT

__alpha-qua.yaml__

    version: 1.1.1

__alpha-prd.yaml__

    version: 1.0.4

Which translates to version 1.0.4 in `prd`, 1.1.1 in `qua` and 1.1.2-SNAPSHOT in `dev`. The deployment specification used to deploy the application would use the `version` property to specify the image tag and in metadata used to identify the pod at runtime:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${app}
  labels:
    app: ${app}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${app}
  template:
    metadata:
      labels:
        app: ${app}
    spec:
      containers:
      - name: ${app}-container
        image: "${container.image}:${version}"
```

## Installation
### Add the CRD's
Add the Custom Resource Definitions to the cluster:
```
kubectl apply -f deploy/crds/cloudconfig_crd.yaml
kubectl apply -f deploy/crds/cloudconfigenv_crd.yaml
```

### Adapt and add the ClusterRole
The `cloud-config-operator` role defined in [deploy/role.yaml] should be adapted to the specific needs of your Kubernetes cluster. The following permissions are required for the basic operation of the `cloud-config-operator`

// TODO tune the rules required for the operation of cloud-config-operator
- watch, list , retrieve, create, update and delete namespaces
  ```yaml
  // TODO add role yaml rules
  ```
In addition the operator needs to have all the required permissions to manage the apps. Typically this means creating, retrieving and deleting deployments but additional rules  may be required if for example the apps define `CronJob`s in their YAML specifications.
```
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/role_binding.yaml
```
### Deploy the operator
Replace the REPLACE_IMAGE placeholder in  deploy/operator.yaml and deploy the operator:
```
sed 's|REPLACE_IMAGE|chrsoo/cloud-config-operator:latest|g' deploy/operator.yaml | kubectl apply -f -
```
## Usage
Synchronziation of a `CloudConfig` application (or list of applications) is started by creating the CR:
```
export NAMESPACE="production"

cat <<<EOF
apiVersion:     k8s.jabberwocky.se/v1alpha1
kind:           CloudConfig
metadata:
  name:         my-app
spec:
  credentials:
    secret:     cloud-config-secret
  profile:      [ prd, us-west ]
  period:       10
EOF | kubectl --namespace ${NAMESPACE} apply -f -
```
To stop synchronization simply delete the CR.

If the `period` is not defined synchronization is only done once and if a value is given synchronization occurs every `period` number of seconds.

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
