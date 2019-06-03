package cloudconfig

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/imdario/mergo"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8s/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new CloudConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCloudConfig{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cloudconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CloudConfig
	err = c.Watch(&source.Kind{Type: &k8v1alpha1.CloudConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CloudConfig
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &k8v1alpha1.CloudConfig{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCloudConfig{}

// ReconcileCloudConfig reconciles a CloudConfig object
type ReconcileCloudConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CloudConfig object and makes changes based on the
// state read and what is in the CloudConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCloudConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	start := time.Now()
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CloudConfig")

	// Fetch the CloudConfig instance
	c := &k8v1alpha1.CloudConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, c)
	if err != nil {
		if k8errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	c = getEffectiveConfig(c)
	if err = validate(&c.Spec); err != nil {
		log.Error(err, "Validation failed")
		// Return and don't requeue
		return reconcile.Result{}, nil
	}

	// Reconcile the CloudConfig
	apps, err := r.reconcileApps(c)
	if err != nil {
		reqLogger.Error(err, "Reconciliation failed")
	} else if len(apps) == 0 {
		reqLogger.Info(fmt.Sprintf("Apps not found for field '%s' of app '%s'", c.Spec.AppList, c.Spec.AppName))
	} else {
		reqLogger.Info(fmt.Sprintf("Reconciled %d app(s) %v in %v", len(apps), apps, time.Since(start)))
	}

	// check if this is a one-off reconciliation
	if c.Spec.Period <= 0 {
		reqLogger.Info("Reconciled CloudConfig; no rescheduling")
		// Don't reschedule as this is a one-off reconciliation
		return reconcile.Result{}, nil
	}

	// reschedule the next reconciliation cycle
	next, skipped := c.GetDurationUntilNextCycle(start)
	if skipped {
		reqLogger.Info("Skipping one or more cycles as reconciliation took too long, consider a prolonging the period!")
	}
	reqLogger.Info(fmt.Sprintf("Reconciled CloudConfig; rescheduling in %v", next))
	return reconcile.Result{Requeue: true, RequeueAfter: next}, nil
}

func getEffectiveConfig(c *k8v1alpha1.CloudConfig) *k8v1alpha1.CloudConfig {
	eff := c.DeepCopy()

	// merge defaults for properties that are not specified
	mergo.Merge(&eff.Spec, k8v1alpha1.NewCloudConfigSpec())
	// special rule for `Period` as ints are not merged
	if eff.Spec.Period == 0 {
		eff.Spec.Period = c.Spec.Period
	}

	fallBackIfEmpty(&eff.Spec.AppName, c.ObjectMeta.Name)

	return eff
}

func (r *ReconcileCloudConfig) reconcileApps(c *k8v1alpha1.CloudConfig) ([]string, error) {
	client, err := r.createClient(c)
	if err != nil {
		return nil, err
	}

	var apps []string
	if c.Spec.AppList == "" {
		// Synchronize a single app
		apps = []string{c.Spec.AppName}
	} else {
		// Synchronize the apps found in the AppList field of the AppName app
		apps, err = client.getApps(c.Spec.AppList, c.Spec.AppName, c.Spec.Label, c.Spec.Profile...)
		if err != nil {
			return nil, err
		}
		// Order alphabetically to maintain consistency when applying the CloudConfig
		sort.Strings(apps)
	}

	// concatenate all app files into one configuration for the entire namespace
	spec := make([]byte, 0, 1024)
	for _, app := range apps {
		file, err := client.GetConfigFile(c.Spec.SpecFile, app, c.Spec.Label, c.Spec.Profile...)
		if err != nil {
			return nil, err
		}
		// TODO ensure that the file is valid YAML before appending it to the spec
		spec = appendYAMLDoc(app, spec, file)
	}

	if len(spec) > 0 {
		err = r.apply(c.Namespace, &spec)
		if err != nil {
			return nil, err
		}
	}
	return apps, nil
}

var execCommand = exec.Command

func (r *ReconcileCloudConfig) apply(namespace string, spec *[]byte) error {
	cmd := execCommand(
		"kubectl",
		"--namespace="+namespace,
		"apply",
		"--prune",
		"--all",
		"-f",
		"-")

	cmd.Stdin = bytes.NewReader(*spec)
	log.Info(strings.Join(cmd.Args, " "))
	var out []byte
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Error(err, fmt.Sprintf("Could not apply spec for '%s'", namespace),
			"command", strings.Join(cmd.Args, " "),
			"output", string(out))
		return err
	}

	log.Info(fmt.Sprintf("Applied spec for '%s'", namespace),
		"command", strings.Join(cmd.Args, " "),
		"output", string(out))
	return nil
}

func (r *ReconcileCloudConfig) createClient(c *k8v1alpha1.CloudConfig) (*CloudConfigClient, error) {

	opts := make([]func(*CloudConfigClient), 0, 10)
	var err error

	if opts, err = r.appendCredentialsOptions(opts, c); err != nil {
		return nil, err
	}

	if opts, err = r.configureTrustStore(opts, c); err != nil {
		return nil, err
	}

	if c.Spec.Insecure {
		opts = append(opts, Insecure())
	}

	return New(c.Spec.Server, opts...)
}

func (r *ReconcileCloudConfig) configureTrustStore(
	opts []func(*CloudConfigClient),
	c *k8v1alpha1.CloudConfig) ([]func(*CloudConfigClient), error) {

	if c.Spec.TrustStore == "" {
		return opts, nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: c.Spec.TrustStore, Namespace: c.Namespace}
	if err := r.client.Get(context.TODO(), name, secret); err != nil {
		return nil, err
	}

	return append(opts, TrustStore(secret.Data)), nil
}

func (r *ReconcileCloudConfig) appendCredentialsOptions(
	opts []func(*CloudConfigClient),
	c *k8v1alpha1.CloudConfig) ([]func(*CloudConfigClient), error) {

	var err error
	cr := &c.Spec.Credentials

	// Configure credentials only if secret has been set
	if cr.Secret == "" {
		return opts, nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: cr.Secret, Namespace: c.Namespace}
	if err := r.client.Get(context.TODO(), name, secret); err != nil {
		return nil, err
	}

	opts, err = appendBearerAuthOption(opts, cr, secret)
	if err != nil {
		return nil, err
	}

	opts, err = appendBasicAuthOption(opts, cr, secret)
	if err != nil {
		return nil, err
	}

	opts, err = appendClientCertOption(opts, cr, secret)
	if err != nil {
		return nil, err
	}

	return opts, nil
}

func appendClientCertOption(
	opts []func(*CloudConfigClient),
	cr *k8v1alpha1.CloudConfigCredentials,
	secret *corev1.Secret) ([]func(*CloudConfigClient), error) {

	cert, hasCert := secret.Data[cr.Cert]
	key, hasKey := secret.Data[cr.Key]

	if hasCert && hasKey {
		tlsOpt := ClientCert(cert, key)
		return append(opts, tlsOpt), nil
	}

	if hasCert != hasKey {
		return nil, fmt.Errorf(
			"both cert('%s') and key('%s') entries have to be defined in the secret '%s' for client certificates",
			cr.Cert, cr.Key, secret.Name)
	}

	return opts, nil
}

func appendBasicAuthOption(
	opts []func(*CloudConfigClient),
	cr *k8v1alpha1.CloudConfigCredentials,
	secret *corev1.Secret) ([]func(*CloudConfigClient), error) {

	username, hasUsername := secret.Data[cr.Username]
	password, hasPassword := secret.Data[cr.Password]

	if hasUsername && hasPassword {
		basicAuthOpt := BasicAuth(string(username), string(password))
		return append(opts, basicAuthOpt), nil
	}

	if hasUsername != hasPassword {
		return nil, fmt.Errorf(
			"both username('%s') and password('%s') entries must be defined in secret '%s' for basic auth",
			cr.Username, cr.Password, secret.Name)
	}

	return opts, nil
}

func appendBearerAuthOption(
	opts []func(*CloudConfigClient),
	cr *k8v1alpha1.CloudConfigCredentials,
	secret *corev1.Secret) ([]func(*CloudConfigClient), error) {

	if token, ok := secret.Data[cr.Token]; ok {
		return append(opts, BearerAuth(string(token))), nil
	}

	return opts, nil
}

func validate(spec *k8v1alpha1.CloudConfigSpec) error {
	validationErrors := field.ErrorList{}
	if spec.Server == "" {
		path := field.NewPath("server")
		fieldErr := field.Required(path, "A Config Server URL must be provided")
		validationErrors = append(validationErrors, fieldErr)
	}

	// Allow plain http only if insecure is true, if no protocol scheme
	// is specified we automatically use https in Environment!
	url := strings.ToLower(spec.Server)
	if !spec.Insecure && strings.HasPrefix(url, "http:") {
		path := field.NewPath("server")
		fieldErr := field.Invalid(path, spec.Server, "URL must use the `https` scheme")
		validationErrors = append(validationErrors, fieldErr)
	}

	if spec.AppName == "" {
		path := field.NewPath("appName")
		fieldErr := field.Invalid(path, spec.AppName, "appName must be specified")
		validationErrors = append(validationErrors, fieldErr)
	}

	if spec.SpecFile == "" {
		path := field.NewPath("specFile")
		fieldErr := field.Invalid(path, spec.SpecFile, "specfile must be specified")
		validationErrors = append(validationErrors, fieldErr)
	}

	if len(validationErrors) > 0 {
		// TODO add CloudConfigSpec's group and kind to groupKind instance
		groupKind := schema.GroupKind{}
		err := k8errors.NewInvalid(groupKind, "CloudConfigSpec", validationErrors)
		// TODO return error instance wrapping err
		return errors.New(err.Error())
		// return errors.New("Validation failed")
	}

	return nil
}

func fallBackIfEmpty(field *string, defaultValue string) {
	if *field == "" {
		*field = defaultValue
	}
}
