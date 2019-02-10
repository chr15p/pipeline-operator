package pipeline

import (
	"context"

	pipelinev1alpha1 "pipeline-operator/pkg/apis/pipeline/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
        //batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var log = logf.Log.WithName("controller_pipeline")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Pipeline Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePipeline{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("pipeline-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Pipeline
	err = c.Watch(&source.Kind{Type: &pipelinev1alpha1.Pipeline{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Pipeline
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &pipelinev1alpha1.Pipeline{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePipeline{}

// ReconcilePipeline reconciles a Pipeline object
type ReconcilePipeline struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Pipeline object and makes changes based on the state read
// and what is in the Pipeline.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePipeline) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Pipeline")

	// Fetch the Pipeline instance
	pipeline := &pipelinev1alpha1.Pipeline{}
	err := r.client.Get(context.TODO(), request.NamespacedName, pipeline)
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

	if pipeline.Spec.Stages == nil {
		reqLogger.Info("Pipeline has no stages defined")
		return reconcile.Result{}, nil
	}

	totalstages := len(pipeline.Spec.Stages)
	stages := make([]corev1.PodPhase,totalstages)
	stagename := make(map[string]int,totalstages)

	for j := range pipeline.Spec.Stages {
		name := pipeline.Spec.Stages[j].Job.ObjectMeta.Name
		stagename[name] = j
		log.Info("status:","Name =",name,"stage =",j)
	}

	podList := &corev1.PodList{}
	lbs := map[string]string{
		"app":     pipeline.Name,
	}
	labelSelector := labels.SelectorFromSet(lbs)
	listOps := &client.ListOptions{Namespace: pipeline.Namespace, LabelSelector: labelSelector}
	if err = r.client.List(context.TODO(), listOps, podList); err != nil {
        	return reconcile.Result{}, err
	}


    	for _, pod := range podList.Items {
		i := stagename[pod.ObjectMeta.Name]
		stages[i] =  pod.Status.Phase
		log.Info("status:","pod.ObjectMeta.Name =",pod.ObjectMeta.Name,"phase =",pod.Status.Phase,"i",i,"value",stages[i])
	}

	flag := 1
	for j := range stages {
		log.Info("stages:","j =",j,"value =",stages[j])
		switch s := stages[j]; s {
		case "":
			if flag == 1 {
				err = r.startPod(pipeline,j)
				if err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, nil
			}
			flag = 0
		case corev1.PodSucceeded:
			flag = 1
		case corev1.PodRunning:
			flag = 0
			return reconcile.Result{}, nil
		case corev1.PodPending: 
			flag = 0
			return reconcile.Result{}, nil
		default:
			flag = 0
			return reconcile.Result{}, nil
		}

	}
	return reconcile.Result{}, nil
}


func (r *ReconcilePipeline) startPod(pipeline *pipelinev1alpha1.Pipeline,j int) error {
	log.Info("Creating a new Pod=","j",j, "Pod.Namespace", pipeline.Namespace, "pipeline.Name", pipeline.Spec.Stages[j].Name)
	pod := newPodForStage(pipeline,j)
	if err := controllerutil.SetControllerReference(pipeline, pod, r.scheme); err != nil {
		return err
 	}

       	err := r.client.Create(context.TODO(), pod)
        if err != nil {
	    log.Error(err, "Failed to create job", "Stage.name", pipeline.Spec.Stages[j].Name)
            return err
	}
	// Pod created successfully - don't requeue
	return nil
}



// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForStage(cr *pipelinev1alpha1.Pipeline,i int) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	pod:= &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.Stages[i].Job.ObjectMeta.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: *cr.Spec.Stages[0].Job.Template.Spec.DeepCopy(),
	}

	//copy(cr.Spec.Stages[0].Job.Template.Spec,pod.Spec)
	return pod
}
