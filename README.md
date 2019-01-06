# Cloud Config Operator
Provision Kubernetes applications using Spring Cloud Config Server
with a GitOps approach.

__This is currently just a skeleton project!__


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
  credentials: cloud-config     # Cloud Config Server secret
  label: master                 # label used for all apps, defaults to 'master'
  specFile: deployment.yaml     # app spec file, defaults to 'deployment.yaml'
  appName: dms-cluster          # application name, defaults to the CloudConfig name
  appList: services             # application list property
  schedule: "*/1 * * * *"       # cron job schedule
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
## Implementation Notes

* For each CloudConfig object the operator maintains a CronJob that reconciles the CloudConfig
* The CronJob runs the Cloud Config Operator Docker image with the `--reconcile --config={...}` parameters
* The value of the `--config` parameter is the JSON serialized CloudConfig object

## TODO

- [x] Apply specification file for each app
- [x] Implement `CloudConfig.Reconcile()`
- [x] Adapt `main()` to support reconciliation (`--reconcile={json}`)
- [ ] Add external dependencies to vendor branches
- [ ] Change Cron Job container to use CloudConfig image

## References
* [Quick Intro to Spring Cloud Config](https://www.baeldung.com/spring-cloud-configuration)
