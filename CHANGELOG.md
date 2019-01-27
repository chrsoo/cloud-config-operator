# v0.2.0 Alpha
- Introduces the `CloudConfigEnv` CRD as dependent object to `CloudConfig`
- The `CloudConfigEnv` CRD represents the environment for a given App or number of Apps
- Support for Bearer, Basic and SSL Client Certificate authentication
- Support for a Trust Store of certificates or a single Root CA
- Still uses `kubectl apply --prune -f -` to align the environment namespace with the Cloud Config configuration

# v0.1.0 First (experimental)
- Experimental release with `CloudConfig` CRD using CronJobs
- Uses `kubectl apply --prune -f -` to align the environment namespace with the Cloud Config configuration
