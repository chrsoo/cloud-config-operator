// Package v1alpha1 contains API Schema definitions for the k8 v1alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=k8.jabberwocky.se
package v1alpha1

/*
Implementation Notes

CloudConfig Controller
- Each CloudConfig object maps to a CronJob that will periodically synchronize the CloudConfiguration
- The actual state is found by enumerating all CronJobs labeled `cloud-config`
*/
