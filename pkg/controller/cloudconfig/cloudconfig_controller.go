package cloudconfig

import (
	"context"

	k8v1alpha1 "github.com/chrsoo/cloud-config-operator/pkg/apis/k8/v1alpha1"
	jobv1 "k8s.io/api/batch/v1"
	cronv1 "k8s.io/api/batch/v1beta1"
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
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new CronJob object
	job := newCronJobForCR(instance)

	// Set CloudConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, job, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this CronJob already exists
	found := &cronv1.CronJob{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
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

// newCronJobForCR returns a CronJob pod with the same name/namespace as the cr
func newCronJobForCR(cr *k8v1alpha1.CloudConfig) *cronv1.CronJob {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &cronv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-cron",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: cronv1.CronJobSpec{
			// TODO use schedule from CR
			Schedule: "*/1 * * * *",
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
							Containers: []corev1.Container{
								{
									Name:    "busybox",
									Image:   "busybox",
									Command: []string{"echo", cr.Name},
								},
							},
							RestartPolicy: "Never",
						},
					},
				},
			},
		},
	}
}
