// Package v1alpha1 contains API Schema definitions for the k8 v1alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=k8.jabberwocky.se
package v1alpha1

/*
Implementation Notes

CloudConfig Controller
- Each CloudConfig object maps to a CronJob that will periodically synchronize the CloudConfiguration
- The actual state is found by enumerating all CronJobs labeled `cloud-config`

CloudConfigServer Controller
- Each CloudConfigServer object maps to a Spring Cloud Config Server Deployment
- Each time the CloudConfigServer object is changed the Deployment spec is updated and applied
- Server YAML configuration is converted to JSON and fed as an argument to the server image `--spring.application.json={...}`

Issues
- How should apps no longer present in a configuration be removed?
  - Manage one namespace per app and delete the namespace?
- How should the Spring Cloud Config Server image be built?

*/
