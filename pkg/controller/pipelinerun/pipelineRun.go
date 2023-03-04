package pipelinerun

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/apoorvajagtap/trackPodCRD/pkg/apis/pipeline/v1alpha1"
	pClientSet "github.com/apoorvajagtap/trackPodCRD/pkg/client/clientset/versioned"
	pInformer "github.com/apoorvajagtap/trackPodCRD/pkg/client/informers/externalversions/pipeline/v1alpha1"
	pLister "github.com/apoorvajagtap/trackPodCRD/pkg/client/listers/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by TrackPod"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "TrackPod synced successfully"
)

// Controller implementation for TrackPod resources
// TODO: Record events
type Controller struct {
	// K8s clientset
	kubeClient kubernetes.Interface
	// things required for controller:
	// - clientset for custom resource
	prunClient pClientSet.Interface
	// - resource (informer) cache has synced
	prunSync cache.InformerSynced
	// - interface provided by informer
	prunLister pLister.PipelineRunLister

	// taskrun specific lisers
	trunLister pLister.TaskRunLister
	// - queue
	// stores the work that has to be processed, instead of performing
	// as soon as it's changed.
	// Helps to ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	wq workqueue.RateLimitingInterface
}

// returns a new TrackPod controller
func NewController(kubeClient kubernetes.Interface, prunClient pClientSet.Interface, prunInformer pInformer.PipelineRunInformer, trunInformer pInformer.TaskRunInformer) *Controller {
	c := &Controller{
		kubeClient: kubeClient,
		prunClient: prunClient,
		prunSync:   prunInformer.Informer().HasSynced,
		prunLister: prunInformer.Lister(),
		trunLister: trunInformer.Lister(),
		wq:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "PipelineRun"),
	}

	// event handler when the pipelineRun resources are added/deleted/updated.
	prunInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.handlePipelineAdd,
			UpdateFunc: func(old, obj interface{}) {
				oldTpod := old.(*v1alpha1.PipelineRun)
				newTpod := obj.(*v1alpha1.PipelineRun)
				if newTpod == oldTpod {
					return
				}
				c.handlePipelineAdd(obj)
			},
			DeleteFunc: c.handlePipelineDel,
		},
	)

	return c
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until ch
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(ch chan struct{}) error {
	defer c.wq.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting the PipelineRun controller")

	// Wait for the caches to be synced before starting workers
	if ok := cache.WaitForCacheSync(ch, c.prunSync); !ok {
		log.Println("failed to wait for cache to sync")
	}
	// Launch the goroutine for workers to process the CR
	klog.Info("Starting workers")
	go wait.Until(c.worker, time.Second, ch)
	klog.Info("Started workers")
	<-ch
	klog.Info("Shutting down the worker")

	return nil
}

// worker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue
func (c *Controller) worker() {
	for c.processNextItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextItem() bool {
	item, shutdown := c.wq.Get()
	if shutdown {
		klog.Info("Shutting down")
		return false
	}

	defer c.wq.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		klog.Errorf("error while calling Namespace Key func on cache for item %s: %s", item, err.Error())
		return false
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		klog.Errorf("error while splitting key into namespace & name: %s", err.Error())
		return false
	}

	prun, err := c.prunLister.PipelineRuns(ns).Get(name)
	if err != nil {
		klog.Errorf("error %s, Getting the prun resource from lister.", err.Error())
		return false
	}

	trun, err := c.syncHandler(prun)
	if err != nil {
		klog.Errorf("Error creating the TaskRun for %s PipelineRun", prun.Name, err.Error())
		return false
	}

	if trun != nil {
		// wait for pods to be ready
		if err := c.waitForPods(trun); err != nil {
			klog.Errorf("error %s, waiting for pods to meet the expected state", err.Error())
		}

		// update taskrun status
		if err := c.updateTrunStatus(trun); err != nil {
			klog.Errorf("error %s updating PipelineRun status", err.Error())
		}

		// update pipelinerun status
		if err = c.updatePrunStatus(prun, trun); err != nil {
			klog.Errorf("error %s updating PipelineRun status", err.Error())
		}

		if prun.Status.Count == c.totalCompletedPods(trun) {
			// fmt.Printf("About to mark trun %v as done\n", trun.Name)
			c.wq.Done(trun)
		}
	}

	return true
}

// syncHandler checks the status of pipeline, if current != desired, meets the requirement.
func (c *Controller) syncHandler(prun *v1alpha1.PipelineRun) (*v1alpha1.TaskRun, error) {
	var createTrun bool

	fmt.Println("starting the check")
	if prun.Spec.Message != prun.Status.Message || prun.Spec.Count != prun.Status.Count {
		fmt.Println("Should have entered check inside syncHandler >>>>>>>>>>>> ")
		createTrun = true
	}

	fmt.Println("reached here >>>>>>>>>>>>>>>>>>>>>>> ", prun.Status, prun.Spec)

	// create the taskrun CR.
	if createTrun {
		fmt.Println("Inside the condition to createTrun")
		trun, err := c.prunClient.AjV1alpha1().TaskRuns(prun.Namespace).Create(context.TODO(), newTaskRun(prun), metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("TaskRun creation failed for Pipeline %s", prun.Name)
			return nil, err
		}

		if trun != nil {
			klog.Infof("Taskrun %s has been created for PipelineRun %s", trun.Name, prun.Name)
			if err := c.createPodTask(prun, trun); err != nil {
				klog.Errorf("Taskrun %s failed to create pods", trun.Name)
			}
		}

		return trun, nil
	}

	return nil, nil
}

// Creates the new pod with the specified template
func newTaskRun(prun *v1alpha1.PipelineRun) *v1alpha1.TaskRun {
	return &v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-trun-%v", prun.Name, prun.ObjectMeta.Generation),
			Namespace: prun.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(prun, v1alpha1.SchemeGroupVersion.WithKind("PipelineRun")),
			},
		},
		Spec: v1alpha1.TaskRunSpec{
			Message: prun.Spec.Message,
			Count:   prun.Spec.Count,
		},
	}
}

// Updates the status section of TrackPod
func (c *Controller) updatePrunStatus(prun *v1alpha1.PipelineRun, trun *v1alpha1.TaskRun) error {

	p, err := c.prunClient.AjV1alpha1().PipelineRuns(prun.Namespace).Get(context.Background(), prun.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	t, err := c.prunClient.AjV1alpha1().TaskRuns(trun.Namespace).Get(context.Background(), trun.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	p.Status.Count = t.Status.Count
	p.Status.Message = t.Status.Message

	_, err = c.prunClient.AjV1alpha1().PipelineRuns(prun.Namespace).UpdateStatus(context.Background(), p, metav1.UpdateOptions{})
	return err
}

func (c *Controller) handlePipelineAdd(obj interface{}) {
	klog.Info("handlePipelineAdd is here!!!")
	c.wq.Add(obj)
}

func (c *Controller) handlePipelineDel(obj interface{}) {
	klog.Info("handlePipelineDel is here!!")
	c.wq.Done(obj)
}
