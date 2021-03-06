package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// CloudConfigSpec defines the desired state of CloudConfig
type CloudConfigSpec struct {
	// Application name, defaults to system name
	AppName string `json:"appName,omitempty"`

	// List or profile names
	Profile []string `json:"profile,omitempty"`

	// label used for all apps, defaults to 'master'
	Label string `json:"label,omitempty"`

	// Cloud Config Server name or URL
	Server string `json:"server,omitempty"`

	// app spec file name, defaults to 'deployment.yaml'
	SpecFile string `json:"specFile,omitempty"`

	// Application list property name, optional
	AppList string `json:"appList,omitempty"`

	// Period is the number of seconds between cloud config synchronizations,
	// a 0 value means that the environment is updated only once after each CloudConfig change
	Period int `json:"period,omitempty"`

	// TrustStore optionally defines the name of a secret containing all trusted certificates
	TrustStore string `json:"trustStore,omitempty"`

	// Cloud Config Server secret containing cloud config credentials, optional
	Credentials CloudConfigCredentials `json:"credentials,omitempty"`

	// If Insecure is 'true' certificates are not required for
	// servers outside the cluster and SSL errors are ignored.
	Insecure bool `json:"insecure,omitempty"`
}

// CloudConfigCredentials contains the metadata used to retrieve a Kubernetes secret containing
// Cloud Config Server credentials.
type CloudConfigCredentials struct {
	// Secret is the name of the secret that holds all credentials, required
	Secret string `json:"secret,omitempty"`
	// Username is the name of the username secret entry, defaults to `username`
	Username string `json:"username,omitempty"`
	// Password is the name of the password secret entry, defaults to `password`
	Password string `json:"password,omitempty"`
	// Token is the name of the token secret entry, defaults to `token`
	Token string `json:"token,omitempty"`
	// Cert is the name of the client certificate secret entry, defaults tp `cert.pem`
	Cert string `json:"cert,omitempty"`
	// Key is the name of the client certificate key secret entry, defaults to `key.pem`
	Key string `json:"key,omitempty"`
	// RootCA is the name of the secret entry for the certificate used to sign the server certificate,
	// defaults to `cert.key`
	RootCA string `json:"rootCA,omitempty"`
}

// GetDurationUntilNextCycle returns the time.Duration until the start of the next reconciliation cycle.
// This is calculated as the Period minus the duration from the start of the current cycle. If the
// current cycle took longer than the period the boolean result is returned as true indicating that
// one or more periods were skipped.
func (c CloudConfig) GetDurationUntilNextCycle(startTime time.Time) (time.Duration, bool) {
	duration := time.Since(startTime)
	period := time.Duration(c.Spec.Period) * time.Second
	if duration < period {
		return period - duration, true
	}
	return period, false
}

// CloudConfigStatus defines the observed state of CloudConfig
type CloudConfigStatus struct {
	// TODO think through how to defined the current status of a CloudConfig CloudConfigSpec
	NamespaceStatus metav1.Status `json:"namsepaceStatus,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudConfig is the Schema for the cloudconfigs API
// +k8s:openapi-gen=true
type CloudConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudConfigSpec   `json:"spec,omitempty"`
	Status CloudConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudConfigList contains a list of CloudConfig
type CloudConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudConfig{}, &CloudConfigList{})
}

// NewCloudConfigSpec returns the default spec for CloudConfig
func NewCloudConfigSpec() *CloudConfigSpec {
	return &CloudConfigSpec{
		Label:       "master",
		Server:      "cloud-config-server:8888",
		SpecFile:    "deployment.yaml",
		Credentials: *NewCloudConfigCredentials(),
	}
}

// NewCloudConfigCredentials returns the default credentials
func NewCloudConfigCredentials() *CloudConfigCredentials {
	return &CloudConfigCredentials{
		Username: "username",
		Password: "password",
		Token:    "token",
		Cert:     "cert.pem",
		Key:      "cert.key",
		RootCA:   "ca.pem",
	}
}
