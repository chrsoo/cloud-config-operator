package cloudconfig

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/chrsoo/cloud-config-operator/version"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8/v1alpha1"
	jobv1 "k8s.io/api/batch/v1"
	cronv1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
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

const (
	// DefaultSchedule is the default cron job reconciliation schedule
	DefaultSchedule = "*/1 * * * *"
	// DefaultCronJobImage is the default Docker image used for the reconcilaition CronJob
	DefaultCronJobImage = "chrsoo/cloud-config-operator:" + version.Version
	// CronJobTimeout defines the timeout before the CronJob is automatically terminated, cf `activeDeadlineSeconds` of the K8 JobSpec
	CronJobTimeout = 300
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

// Reconcile reads that state of the cluster for a CloudConfig object and makes changes based on the state read
// and what is in the CloudConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCloudConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CloudConfig")

	// Fetch the CloudConfig instance
	instance := &k8v1alpha1.CloudConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	if err := validate(&instance.Spec); err != nil {
		log.Error(err, "Validation failed")
		// Return and don't requeue
		return reconcile.Result{}, nil
	}

	// Secrets are configurable per environment but we only have one CronJob. Currently
	// a globally defined secret is assumed and is associated with the one CronJob managing
	// all environments.
	// Using multiple CronJobs per environment can also be motivated by additional control and improved
	// isolation.
	// TODO create one CronJob per environment to improve isolation and enable different secrets per environment

	// Define a new CronJob object
	job, err := newCronJobForCR(instance)
	if err != nil {
		reqLogger.Error(err, "Could not marshal CloudConfig as JSON; returning request to queue")
		return reconcile.Result{}, err
	}

	job = r.configureSecret(instance, job)

	// Set CloudConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, job, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this CronJob already exists
	found := &cronv1.CronJob{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && k8errors.IsNotFound(err) {
		reqLogger.Info("Creating a new CronJob", "CronJob.Namespace", job.Namespace, "CronJob.Name", job.Name)
		err = r.client.Create(context.TODO(), job)
		if err != nil {
			return reconcile.Result{}, err
		}

		// CronJob created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// CronJob already exists - don't requeue
	reqLogger.Info(
		"Skip reconcile: CronJob already exists",
		"CronJob.Namespace", found.Namespace, "CronJob.Name", found.Name)
	return reconcile.Result{}, nil
}

func setFallbackValues(cr *k8v1alpha1.CloudConfig) {
	fallBackIfEmpty(&cr.Spec.Environment.Name, cr.ObjectMeta.GetName())
	fallBackIfEmpty(&cr.Spec.Environment.AppName, cr.Spec.Environment.Name)
	fallBackIfEmpty(&cr.Spec.Environment.Key, cr.Spec.Environment.Name)
	fallBackIfEmpty(&cr.Spec.Schedule, DefaultSchedule)
}

func fallBackIfEmpty(field *string, defaultValue string) {
	if *field == "" {
		*field = defaultValue
	}
}

// newCronJobForCR returns a CronJob pod with the same name/namespace as the cr
func newCronJobForCR(cr *k8v1alpha1.CloudConfig) (*cronv1.CronJob, error) {
	setFallbackValues(cr)
	spec, err := json.Marshal(cr.Spec)
	if err != nil {
		return nil, err
	}
	config := string(spec)

	labels := map[string]string{
		"app": cr.Name,
	}

	job := &cronv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-cron",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: cronv1.CronJobSpec{
			Schedule:          cr.Spec.Schedule,
			ConcurrencyPolicy: cronv1.ForbidConcurrent,
			JobTemplate: cronv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cr.Name + "-job",
					Namespace: cr.Namespace,
					Labels:    labels,
				},
				Spec: jobv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      cr.Name + "-pod",
							Namespace: cr.Namespace,
							Labels:    labels,
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: "cloud-config-operator",
							// TODO make the ActiveDeadlineSeconds configurable
							ActiveDeadlineSeconds: timeout(),
							Containers: []corev1.Container{
								{
									Name:    "cloud-config-operator",
									Image:   DefaultCronJobImage,
									Command: []string{"cloud-config-operator", "--reconcile", config},
									// FIXME change back to ImagePullPolicy: Always
									ImagePullPolicy: "Never",
								},
							},
							RestartPolicy: "Never",
						},
					},
				},
			},
		},
	}
	return job, nil
}

func (r *ReconcileCloudConfig) configureSecret(cr *k8v1alpha1.CloudConfig, job *cronv1.CronJob) *cronv1.CronJob {
	s := cr.Spec.Secret
	if s == "" {
		return job
	}

	var secretPath string
	if strings.Contains(s, "/") {
		secretPath = s
	} else {
		if !r.isSecret(job.Namespace, s) {
			panic("Could not configure secret '" + s + "' for CloudConfig '" + cr.Name + "'")
		}
		secretPath = k8v1alpha1.SecretPathPrefix + s
	}

	job.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "cloud-config-secret",
			MountPath: secretPath,
			ReadOnly:  true,
		},
	}
	job.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "cloud-config-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cr.Spec.Secret,
				},
			},
		},
	}
	return job
}

func (r *ReconcileCloudConfig) isSecret(namespace, name string) bool {
	var secret corev1.Secret
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &secret); err != nil {
		log.Error(err, "Could not get the secret '"+namespace+"."+name+"'")
		if k8errors.IsNotFound(err) {
			return false
		}
		// panic if the error is anything but not found!
		panic(err)
	}
	// we found the secret
	return true
}

func timeout() *int64 {
	t := int64(CronJobTimeout)
	return &t
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
