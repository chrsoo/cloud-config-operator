package cloudconfigenv

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CloudConfigEnv Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCloudConfigEnv{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cloudconfigenv-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CloudConfigEnv
	err = c.Watch(&source.Kind{Type: &k8v1alpha1.CloudConfigEnv{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CloudConfigEnv
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &k8v1alpha1.CloudConfigEnv{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCloudConfigEnv{}

// ReconcileCloudConfigEnv reconciles a CloudConfigEnv object
type ReconcileCloudConfigEnv struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CloudConfigEnv object and makes changes based on the
// state read and what is in the CloudConfigEnv.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCloudConfigEnv) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	start := time.Now()
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CloudConfigEnv")

	// Fetch the CloudConfigEnv instance
	env := &k8v1alpha1.CloudConfigEnv{}
	err := r.client.Get(context.TODO(), request.NamespacedName, env)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	reqLogger = reqLogger.WithValues("environment", env.Name)

	// Reconcile the CloudConfigEnv in the namespace
	if err = r.reconcileNamespace(env, reqLogger); err != nil {
		// Retry as soon as possible
		return reconcile.Result{}, err
	}

	// Reconcile the CloudConfigEnv in the namespace
	apps, err := r.reconcileApps(env)
	if err != nil {
		reqLogger.Error(err, "Reconciliation failed")
	} else if len(apps) == 0 {
		reqLogger.Info(fmt.Sprintf("Apps not found for field '%s' of app '%s'", env.Spec.AppList, env.Spec.AppName))
	} else {
		reqLogger.Info(fmt.Sprintf("Reconciled %d app(s) %v in %v", len(apps), apps, time.Since(start)))
	}

	// Don't reschedule as this is a one-off reconciliation
	if env.Spec.Period == -1 {
		reqLogger.Info("Reconciled CloudConfigEnv; no rescheduling")
		return reconcile.Result{}, nil
	}

	// reschedule the next reconciliation cycle
	next, skipped := env.GetDurationUntilNextCycle(start)
	if skipped {
		reqLogger.Info("Skipping one or more cycles as reconciliation took too long, consider a prolonging the period!")
	}
	reqLogger.Info(fmt.Sprintf("Reconciled CloudConfigEnv; rescheduling in %v", next))
	return reconcile.Result{Requeue: true, RequeueAfter: next}, nil
}

func (r *ReconcileCloudConfigEnv) reconcileApps(env *k8v1alpha1.CloudConfigEnv) ([]string, error) {
	client, err := r.createClient(env)
	if err != nil {
		return nil, err
	}

	var apps []string
	if env.Spec.AppList == "" { // Synchronize a single app
		apps = []string{env.Spec.AppName}
	} else { // Synchronize the apps found in the AppList field of the AppName app
		apps, err = client.getApps(env.Spec.AppList, env.Spec.AppName, env.Spec.Label, env.Spec.Profile...)
		if err != nil {
			return nil, err
		}
		// Get the apps and order alphabetically to maintain consistency when applying the k8Config
		sort.Strings(apps)
	}

	// concatenate all app files into one configuration for the entire namespace
	spec := make([]byte, 0, 1024)
	for _, app := range apps {
		file, err := client.GetConfigFile(env.Spec.SpecFile, app, env.Spec.Label, env.Spec.Profile...)
		if err != nil {
			return nil, err
		}
		// TODO ensure that the file is valid YAML before appending it to the spec
		spec = appendYAMLDoc(app, spec, file)
	}

	if len(spec) > 0 {
		err = r.apply(env.Name, &spec)
		if err != nil {
			return nil, err
		}
	}
	return apps, nil
}

var execCommand = exec.Command

func (r *ReconcileCloudConfigEnv) apply(namespace string, spec *[]byte) error {
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

func (r *ReconcileCloudConfigEnv) createClient(env *k8v1alpha1.CloudConfigEnv) (*CloudConfigClient, error) {

	opts := make([]func(*CloudConfigClient), 0, 10)
	var err error

	if opts, err = r.appendCredentialsOptions(opts, env); err != nil {
		return nil, err
	}

	if opts, err = r.configureTrustStore(opts, env); err != nil {
		return nil, err
	}

	if env.Spec.Insecure {
		opts = append(opts, Insecure())
	}

	return New(env.Spec.Server, opts...)
}

func (r *ReconcileCloudConfigEnv) configureTrustStore(
	opts []func(*CloudConfigClient),
	env *k8v1alpha1.CloudConfigEnv) ([]func(*CloudConfigClient), error) {

	if env.Spec.TrustStore == "" {
		return opts, nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: env.Spec.TrustStore, Namespace: env.Namespace}
	if err := r.client.Get(context.TODO(), name, secret); err != nil {
		return nil, err
	}

	return append(opts, TrustStore(secret.Data)), nil
}

func (r *ReconcileCloudConfigEnv) appendCredentialsOptions(
	opts []func(*CloudConfigClient),
	env *k8v1alpha1.CloudConfigEnv) ([]func(*CloudConfigClient), error) {

	var err error
	cr := &env.Spec.Credentials

	// Configure credentials only if secret has been set
	if cr.Secret == "" {
		return opts, nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: cr.Secret, Namespace: env.Namespace}
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

func (r *ReconcileCloudConfigEnv) reconcileNamespace(
	env *k8v1alpha1.CloudConfigEnv, logger logr.Logger) error {

	// Define a new Namespace for the environment
	ns := newEnvironmentNamespace(env)

	// Set CloudConfigEnv instance as the owner and controller
	if err := controllerutil.SetControllerReference(env, ns, r.scheme); err != nil {
		return err
	}

	// Check if this Namespace already exists
	found := &corev1.Namespace{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Creating a namespace", "Name", ns.Name)
			err = r.client.Create(context.TODO(), ns)
			if err != nil {
				return err
			}
		}
	} else if found.ObjectMeta.Labels["app"] != ns.Labels["app"] || found.ObjectMeta.Labels["env"] != ns.Labels["env"] || found.ObjectMeta.Labels["sys"] != ns.Labels["sys"] {
		found.Labels["app"] = ns.Labels["app"]
		found.Labels["env"] = ns.Labels["env"]
		found.Labels["sys"] = ns.Labels["sys"]
		err = r.client.Update(context.TODO(), found)
	}

	return err // nil if all went well
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newEnvironmentNamespace(cr *k8v1alpha1.CloudConfigEnv) *corev1.Namespace {
	labels := map[string]string{
		"app": cr.Spec.AppName,
		"sys": cr.Spec.Sys,
		"env": cr.Spec.Env,
	}
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cr.Name,
			Labels: labels,
		},
		Spec: corev1.NamespaceSpec{},
	}
}
