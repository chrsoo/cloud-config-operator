# Development
Notes on `cloud-config-operator` development and its implementation

## Versioning
Versioning is based on [SemVer 2.0.0](https://semver.org/) but please note that releases are commuicated using major/minor versions, cf. the [roadmap section](README.md#Roadmap) in the README!

## Implementation

* For each CloudConfig object the operator maintains a CronJob that reconciles the CloudConfig
* The CronJob runs the Cloud Config Operator Docker image with the `--reconcile --config={...}` parameters
* The value of the `--config` parameter is the JSON serialized CloudConfig object