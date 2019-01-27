package cloudconfigenv

import (
	"bytes"
	"encoding/base64"
	"os"
	"os/exec"
	"testing"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

// TODO move to cloudconfg_test.go
func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TODO move to cloudconfg_test.go
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// TODO capture input from stdin
	// FIXME broken mock logic for command line testing
	if os.Args[4] == "get" {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestExecWithStdin(t *testing.T) {
	cmd := execCommand("cat")
	cmd.Stdin = bytes.NewReader([]byte("Hello World!"))
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err, "Command should execute without error")
	assert.NotNil(t, out, "Expected some output")
	assert.Equal(t, "Hello World!", string(out), "Expected output to equal input on stdin")
}

func TestAppendBearerAuthOption(t *testing.T) {
	var err error
	secret := &corev1.Secret{
		Data: map[string][]byte{},
	}
	opts := make([]func(*CloudConfigClient), 0, 1)
	cr := k8v1alpha1.DefaultCloudConfigCredentials()

	opts, err = appendBearerAuthOption(opts, cr, secret)
	assert.NoError(t, err)
	assert.Len(t, opts, 0, "there should not be an option if both username and password are empty")

	token := base64.StdEncoding.EncodeToString([]byte("TOKEN_VALUE"))

	secret.Data["token"] = []byte(token)
	opts, err = appendBearerAuthOption(opts, cr, secret)
	assert.NoError(t, err)
	assert.Len(t, opts, 1, "bearer auth should be configured for token secret entry")
}

func TestAppendBasicAuthOption(t *testing.T) {
	var err error
	secret := &corev1.Secret{
		Data: map[string][]byte{},
	}
	opts := make([]func(*CloudConfigClient), 0, 1)
	cr := k8v1alpha1.DefaultCloudConfigCredentials()

	opts, err = appendBasicAuthOption(opts, cr, secret)
	assert.NoError(t, err)
	assert.Len(t, opts, 0, "there should not be an option if both username and password are empty")

	username := base64.StdEncoding.EncodeToString([]byte("anonymous"))

	secret.Data["username"] = []byte(username)
	opts, err = appendBasicAuthOption(opts, cr, secret)
	assert.Error(t, err, "username requires a password")
	assert.Nil(t, opts)

	password := base64.StdEncoding.EncodeToString([]byte("secret"))

	secret.Data["password"] = []byte(password)
	opts, err = appendBasicAuthOption(opts, cr, secret)
	assert.NoError(t, err)
	assert.Len(t, opts, 1, "basic auth should be configured for username and password")

	delete(secret.Data, "username")
	opts = make([]func(*CloudConfigClient), 0, 1)
	opts, err = appendBasicAuthOption(opts, cr, secret)
	assert.Error(t, err, "Password requires a username")
	assert.Nil(t, opts)

	secret.Data["username"] = []byte("anonymous")
	secret.Data["password"] = []byte(password)
	opts, err = appendBasicAuthOption(opts, cr, secret)
	assert.Error(t, err, "username not Base64 encoded")

	secret.Data["username"] = []byte(username)
	secret.Data["password"] = []byte("secret")
	opts, err = appendBasicAuthOption(opts, cr, secret)
	assert.Error(t, err, "password not Base64 encoded")

}

func TestAppendClientCertOption(t *testing.T) {
	var err error
	cr := k8v1alpha1.DefaultCloudConfigCredentials()
	opts := make([]func(*CloudConfigClient), 0, 1)

	secret := &corev1.Secret{
		Data: map[string][]byte{},
	}

	opts, err = appendClientCertOption(opts, cr, secret)
	assert.NoError(t, err)
	assert.Len(t, opts, 0, "there should not be an option if client cert and keys are not a secret entries")

	key := base64.StdEncoding.EncodeToString([]byte(testClientKey))

	secret.Data["cert.key"] = []byte(key)
	opts, err = appendClientCertOption(opts, cr, secret)
	assert.Error(t, err, "cert.key requires a cert.pem entry")
	assert.Nil(t, opts)

	cert := base64.StdEncoding.EncodeToString([]byte(testClientPem))

	secret.Data["cert.pem"] = []byte(cert)
	opts, err = appendClientCertOption(opts, cr, secret)
	assert.NoError(t, err)
	assert.Len(t, opts, 1, "client certificate should be configured for cert.pem and cert.key entries")

	delete(secret.Data, "cert.key")
	opts = make([]func(*CloudConfigClient), 0, 1)
	opts, err = appendClientCertOption(opts, cr, secret)
	assert.Error(t, err, "cert.pem requires a cert.key entry")
	assert.Nil(t, opts)

	secret.Data["cert.key"] = []byte(testClientKey)
	secret.Data["cert.pem"] = []byte(cert)
	opts, err = appendClientCertOption(opts, cr, secret)
	assert.Error(t, err, "cert.key not Base64 encoded")

	secret.Data["cert.key"] = []byte(testClientKey)
	secret.Data["cert.pem"] = []byte(testClientPem)
	opts, err = appendClientCertOption(opts, cr, secret)
	assert.Error(t, err, "cert.pem not Base64 encoded")
}

func TestAppendTrustStoreOption(t *testing.T) {
	var err error
	opts := make([]func(*CloudConfigClient), 0, 1)

	// we use the client.pem just to have more than one test cert!
	clientPem := base64.StdEncoding.EncodeToString([]byte(testRootCAPem))
	caPem := base64.StdEncoding.EncodeToString([]byte(testClientPem))

	certs := map[string][]byte{
		"ca.pem":     []byte(caPem),
		"client.pem": []byte(clientPem),
	}

	opts, err = appendTrustStoreOption(opts, certs)
	assert.NoError(t, err)
	assert.Len(t, opts, 1, "trust store not configured")

	// TODO assert that trust store certificates have been configured for the http client
}
