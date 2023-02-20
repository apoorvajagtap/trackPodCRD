package controller

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	v1 "github.com/apoorvajagtap/trackPodCRD/pkg/apis/aj.com/v1"
	tClientSet "github.com/apoorvajagtap/trackPodCRD/pkg/client/clientset/versioned"
	tInformer "github.com/apoorvajagtap/trackPodCRD/pkg/client/informers/externalversions/aj.com/v1"
	tLister "github.com/apoorvajagtap/trackPodCRD/pkg/client/listers/aj.com/v1"
	"github.com/kanisterio/kanister/pkg/poll"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	tpodClient tClientSet.Interface
	// - resource (informer) cache has synced
	tpodSync cache.InformerSynced
	// - interface provided by informer
	tpodlister tLister.TrackPodLister
	// - queue
	// stores the work that has to be processed, instead of performing
	// as soon as it's changed.
	// Helps to ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	wq workqueue.RateLimitingInterface
}

// returns a new TrackPod controller
func NewController(kubeClient kubernetes.Interface, tpodClient tClientSet.Interface, tpodInformer tInformer.TrackPodInformer) *Controller {
	c := &Controller{
		kubeClient: kubeClient,
		tpodClient: tpodClient,
		tpodSync:   tpodInformer.Informer().HasSynced,
		tpodlister: tpodInformer.Lister(),
		wq:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "TrackPod"),
	}

	// event handler when the trackPod resources are added/deleted/updated.
	tpodInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.handleAdd,
			UpdateFunc: func(old, obj interface{}) {
				oldTpod := old.(*v1.TrackPod)
				newTpod := obj.(*v1.TrackPod)
				if newTpod == oldTpod {
					return
				}
				c.handleAdd(obj)
			},
			DeleteFunc: c.handleDel,
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
	klog.Info("Starting the TrackPod controller")

	// Wait for the caches to be synced before starting workers
	if ok := cache.WaitForCacheSync(ch, c.tpodSync); !ok {
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

	tpod, err := c.tpodlister.TrackPods(ns).Get(name)
	if err != nil {
		klog.Errorf("error %s, Getting the tpod resource from lister.", err.Error())
		return false
	}

	// filter out if required pods are already available or not:
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"controller": tpod.Name,
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	// TODO: Prefer using podLister to reduce the call to K8s API.
	pList, _ := c.kubeClient.CoreV1().Pods(tpod.Namespace).List(context.TODO(), listOptions)

	if err := c.syncHandler(tpod, pList); err != nil {
		klog.Errorf("Error while syncing the current vs desired state for TrackPod %v: %v\n", tpod.Name, err.Error())
		return false
	}

	// wait for pods to be ready
	err = c.waitForPods(tpod, pList)
	if err != nil {
		klog.Errorf("error %s, waiting for pods to meet the expected state", err.Error())
	}

	// fmt.Println("Calling update status again!!")
	err = c.updateStatus(tpod, tpod.Spec.Message, pList)
	if err != nil {
		klog.Errorf("error %s updating status after waiting for Pods", err.Error())
	}

	return true
}

// total number of 'Running' pods
func (c *Controller) totalRunningPods(tpod *v1.TrackPod) int {
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"controller": tpod.Name,
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	// TODO: Prefer using podLister to reduce the call to K8s API.
	pList, _ := c.kubeClient.CoreV1().Pods(tpod.Namespace).List(context.TODO(), listOptions)

	runningPods := 0
	for _, pod := range pList.Items {
		if pod.ObjectMeta.DeletionTimestamp.IsZero() && pod.Status.Phase == "Running" {
			runningPods++
		}
	}
	return runningPods
}

// syncHandler monitors the current state & if current != desired,
// tries to meet the desired state.
func (c *Controller) syncHandler(tpod *v1.TrackPod, pList *corev1.PodList) error {
	var podCreate, podDelete bool
	iterate := tpod.Spec.Count
	deleteIterate := 0
	runningPods := c.totalRunningPods(tpod)

	if runningPods != tpod.Spec.Count || tpod.Spec.Message != tpod.Status.Message {
		if tpod.Spec.Message != tpod.Status.Message {
			klog.Warningf("the message of TrackPod %v resource has been modified, recreating the pods\n", tpod.Name)
			podCreate = true
			iterate = tpod.Spec.Count
			if runningPods > 0 {
				podDelete = true
				deleteIterate = c.totalRunningPods(tpod)
			}
		} else {
			klog.Warningf("detected mismatch of replica count for CR %v >> expected: %v & have: %v\n\n", tpod.Name, tpod.Spec.Count, runningPods)
			if runningPods < tpod.Spec.Count {
				podCreate = true
				iterate = tpod.Spec.Count - runningPods
				klog.Infof("Creating %v new pods\n", iterate)
			} else if runningPods > tpod.Spec.Count {
				podDelete = true
				deleteIterate = runningPods - tpod.Spec.Count
				klog.Infof("Deleting %v extra pods\n", deleteIterate)
			}

		}
	}

	// Delete extra pod
	// TODO: Detect the manually created pod, and delete that specific pod.
	if podDelete {
		for i := 0; i < deleteIterate; i++ {
			err := c.kubeClient.CoreV1().Pods(tpod.Namespace).Delete(context.TODO(), pList.Items[i].Name, metav1.DeleteOptions{})
			if err != nil {
				klog.Errorf("Pod deletion failed for CR %v\n", tpod.Name)
				return err
			}
		}
	}

	// Creates pod
	if podCreate {
		for i := 0; i < iterate; i++ {
			nPod, err := c.kubeClient.CoreV1().Pods(tpod.Namespace).Create(context.TODO(), newPod(tpod), metav1.CreateOptions{})
			if err != nil {
				if errors.IsAlreadyExists(err) {
					// retry (might happen when the same named pod is created again)
					iterate++
				} else {
					klog.Errorf("Pod creation failed for CR %v\n", tpod.Name)
					return err
				}
			}
			if nPod.Name != "" {
				klog.Infof("Pod %v created successfully!\n", nPod.Name)
			}
		}
	}

	return nil
}

// Creates the new pod with the specified template
func newPod(tpod *v1.TrackPod) *corev1.Pod {
	labels := map[string]string{
		"controller": tpod.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      fmt.Sprintf(tpod.Name + "-" + strconv.Itoa(rand.Intn(10000000))),
			Namespace: tpod.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tpod, v1.SchemeGroupVersion.WithKind("TrackPod")),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "static-nginx",
					Image: "nginx:latest",
					Env: []corev1.EnvVar{
						{
							Name:  "MESSAGE",
							Value: tpod.Spec.Message,
						},
					},
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						"while true; do echo '$(MESSAGE)'; sleep 100; done",
					},
				},
			},
		},
	}
}

// If the pod doesn't switch to a running state within 10 minutes, shall report.
func (c *Controller) waitForPods(tpod *v1.TrackPod, pList *corev1.PodList) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		runningPods := c.totalRunningPods(tpod)
		// fmt.Println("Inside waitforPods ???? totalrunningPods >>>> ", runningPods)

		if runningPods == tpod.Spec.Count {
			return true, nil
		}
		return false, nil
	})
}

// Updates the status section of TrackPod
func (c *Controller) updateStatus(tpod *v1.TrackPod, progress string, pList *corev1.PodList) error {
	t, err := c.tpodClient.AjV1().TrackPods(tpod.Namespace).Get(context.Background(), tpod.Name, metav1.GetOptions{})
	trunningPods := c.totalRunningPods(tpod)
	if err != nil {
		return err
	}

	t.Status.Count = trunningPods
	t.Status.Message = progress
	_, err = c.tpodClient.AjV1().TrackPods(tpod.Namespace).UpdateStatus(context.Background(), t, metav1.UpdateOptions{})

	return err
}

func (c *Controller) handleAdd(obj interface{}) {
	klog.Info("handleAdd is here!!!")
	c.wq.Add(obj)
}

func (c *Controller) handleDel(obj interface{}) {
	klog.Info("handleDel is here!!")
	c.wq.Done(obj)
}
