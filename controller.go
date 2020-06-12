package deploypostgres

import (
	"context"

	postgresv1alpha1 "github.com/talat-shaheen/postgres-go/pkg/apis/postgres/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"reflect"
)

var log = logf.Log.WithName("controller_deploypostgres")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new DeployPostgres Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDeployPostgres{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("deploypostgres-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource DeployPostgres
	err = c.Watch(&source.Kind{Type: &postgresv1alpha1.DeployPostgres{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner DeployPostgres
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &postgresv1alpha1.DeployPostgres{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDeployPostgres implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDeployPostgres{}

// ReconcileDeployPostgres reconciles a DeployPostgres object
type ReconcileDeployPostgres struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a DeployPostgres object and makes changes based on the state read
// and what is in the DeployPostgres.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDeployPostgres) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling DeployPostgres")

	// Fetch the DeployPostgres instance
	instance := &postgresv1alpha1.DeployPostgres{}
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
		// instance created successfully
		currentStatus:="DeployPostgres instance created"
		if !reflect.DeepEqual(currentStatus, instance.Status.Status) {
		instance.Status.Status=currentStatus
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{},err
		}}
	//log.Printf("Reconciling Influxdb Persistent Volume Claim\n")
	// Reconcile the persistent volume claim
	err = r.reconcilePVC(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

// Check if the Deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Deployment
		dep := r.deploymentForPostgres(instance)
		reqLogger.Info("Creating a new Deployment.", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment.", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		// NOTE: that the requeue is made with the purpose to provide the deployment object for the next step to ensure the deployment size is the same as the spec.
		// Also, you could GET the deployment object again instead of requeue if you wish. See more over it here: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/reconcile#Reconciler
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment.")
		return reconcile.Result{}, err
	}

		// deployment created successfully - don't requeue
		currentStatus="Deployment created"
		if !reflect.DeepEqual(currentStatus, instance.Status.Status) {
		instance.Status.Status=currentStatus
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{},err
		}
	}
	// Ensure the deployment size is the same as the spec
	size := instance.Spec.Size
	if *deployment.Spec.Replicas != size {
		deployment.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

// reconcilePVC ensures the Persistent Volume Claim is created.
func (r *ReconcileDeployMysql) reconcilePVC(cr *mysqlv1alpha1.DeployMysql) error {
	// Check if this Persitent Volume Claim already exists
	found := &corev1.PersistentVolumeClaim{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cr.Spec.PersistentVolumeClaim.Name, Namespace: cr.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		//log.Printf("Creating a new Persistent Volume Claim %s/%s", cr.Namespace, cr.Spec.PvcName)
		pvc := newPVCs(cr)

		// Set InfluxDB instance as the owner and controller
		if err = controllerutil.SetControllerReference(cr, pvc, r.scheme); err != nil {
		//	log.Printf("%v", err)
			return err
		}

		err = r.client.Create(context.TODO(), pvc)
		if err != nil {
		//	log.Printf("%v", err)
			return err
		}

		// Persitent Volume Claim created successfully
		currentStatus:="PVC created"
		if !reflect.DeepEqual(currentStatus, cr.Status.Status) {
		cr.Status.Status=currentStatus
		if err := r.client.Status().Update(context.TODO(), cr); err != nil {
			return err
		}}

		return nil
	} else if err != nil {
	//	log.Printf("%v", err)
		return err
	}

	//log.Printf("Skip reconcile: Persistent Volume Claim %s/%s already exists", found.Namespace, found.Name)
	return nil
}
// newPVCs creates the PVCs used by the application.
func newPVCs(cr *mysqlv1alpha1.DeployMysql) *corev1.PersistentVolumeClaim {
	labels := map[string]string{
		"app": cr.Name,
	}

	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.PersistentVolumeClaim.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: cr.Spec.PersistentVolumeClaim.Spec,
	}
}


func (r *ReconcileDeployMysql) deploymentForMysql(m *mysqlv1alpha1.DeployMysql) *appsv1.Deployment {
	//ls := labelsForMemcached(m.Name)
	ls := map[string]string{
		"app": m.Name,
	}
	replicas := m.Spec.Size
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "mysql-go",
					//ServiceAccount: "postgres-go",
					Containers: []corev1.Container{
					 {
                                        //Command: []string{"date", ""},
                                        Name: "mysql",
                                        Image: "mysql:8",
                                        //Image: "postgres",

                                        Ports: []corev1.ContainerPort{{
                                                        ContainerPort: 3306,
                                                        Name:          "mysql",
                                                }},
                                        Env: []corev1.EnvVar{
                                                {Name: "MDATA", Value: "/var/lib/postgresql/data/mdata"},
                                                {Name: "MYSQL_PASSWORD", ValueFrom: &corev1.EnvVarSource{
                        ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "cm"}, Key: "MYSQL_PASSWORD"}},},
                                        //      {Name: "DB_PASSWORD",ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "mongodb"}, Key: "mongodb-root-password"}},},
                                                },
                                        VolumeMounts: []corev1.VolumeMount{{
                                                Name:      "mysql-data",
                                                MountPath: "/var/lib/mysql/data",
                                                }},
                                },

					},
			Volumes: []corev1.Volume{{Name: "mysql-data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName:   m.Spec.PersistentVolumeClaim.Name,},	},}},
					
				},
			},
		},
	}
	// Set instance as the owner of the Deployment.
	controllerutil.SetControllerReference(m, dep, r.scheme)
	return dep
}
