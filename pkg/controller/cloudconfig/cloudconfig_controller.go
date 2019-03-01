package cloudconfig

import (
	"context"
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8s/v1alpha1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_cloudconfig")

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

	// TODO(User) Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CloudConfig
	/* 	err = c.Watch(&source.Kind{Type: &cronv1.CronJob{}}, &handler.EnqueueRequestForOwner{
	   		IsController: true,
	   		OwnerType:    &k8v1alpha1.CloudConfig{},
	   	})
	   	if err != nil {
	   		return err
	   	}
	*/
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

// Reconcile reads that state of the cluster for a CloudConfig object and makes changes based on the state read
// and what is in the CloudConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCloudConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CloudConfig")

	// Fetch the CloudConfig instance
	cloudConfig := &k8v1alpha1.CloudConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cloudConfig)
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

	if err := validate(&cloudConfig.Spec); err != nil {
		log.Error(err, "Validation failed")
		// Return and don't requeue
		return reconcile.Result{}, nil
	}

	// Ensure that all CloudConfigEnvs exists
	for key := range cloudConfig.Spec.Environments {
		env := cloudConfig.GetEnvironment(key)
		if err := r.reconcileEnvironment(cloudConfig, env); err != nil {
			return reconcile.Result{}, err
		}
	}
	// Remove all environments that should not exist
	for _, ref := range cloudConfig.OwnerReferences {
		if !*ref.Controller && !cloudConfig.IsManagedNamespace(ref.Name) {
			env := &k8v1alpha1.CloudConfigEnv{}
			err = r.client.Get(
				context.TODO(),
				types.NamespacedName{Name: ref.Name, Namespace: cloudConfig.Namespace},
				env,
			)
			if err != nil {
				// Continue if the environment is not found else try again later
				if k8errors.IsNotFound(err) {
					continue
				}
				return reconcile.Result{}, err
			}
			// Make sure all owned objects are deleted before the environment itself is deleted
			// TODO find out what happens if end up here again before the environment is deleted
			r.client.Delete(context.TODO(), env, client.PropagationPolicy(metav1.DeletePropagationForeground))
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCloudConfig) reconcileEnvironment(
	config *k8v1alpha1.CloudConfig, env *k8v1alpha1.CloudConfigEnv) error {

	// Set CloudConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(config, env, r.scheme); err != nil {
		return err
	}

	// Check if this Environment already exists
	found := &k8v1alpha1.CloudConfigEnv{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: env.Name, Namespace: env.Namespace}, found)
	if err != nil && k8errors.IsNotFound(err) {
		// It does not, let's create it
		log.Info("Creating a new Environment", "Name", env.Name)
		// If not successfull err is return at the end
		err = r.client.Create(context.TODO(), env)
	} else if err == nil {
		// TODO only update if env and found Specs are different
		env.Spec.DeepCopyInto(&found.Spec)
		err = r.client.Update(context.TODO(), found)
	}
	return err
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

	if len(validationErrors) > 0 {
		// TODO add CloudConfigSpec's group and kind to groupKind instance
		groupKind := schema.GroupKind{}
		err := k8errors.NewInvalid(groupKind, "CloudCongigSpec", validationErrors)
		// TODO return error instance wrapping err
		return errors.New(err.Error())
		// return errors.New("Validation failed")
	}

	return nil
}
