package controller

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	v1 "github.com/apoorvajagtap/trackPodCRD/pkg/apis/aj.com/v1"
	klientset "github.com/apoorvajagtap/trackPodCRD/pkg/client/clientset/versioned"
	kInformer "github.com/apoorvajagtap/trackPodCRD/pkg/client/informers/externalversions/aj.com/v1"
	klientLister "github.com/apoorvajagtap/trackPodCRD/pkg/client/listers/aj.com/v1"
	"github.com/kanisterio/kanister/pkg/poll"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Foo"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Foo synced successfully"
)

type Controller struct {
	// K8s clientset
	client kubernetes.Interface
	// things required for controller:
	// - clientset for custom resource
	klient klientset.Interface
	// - resource (informer) cache has synced
	klusterSync cache.InformerSynced
	// - interface provided by informer
	klister klientLister.TrackPodLister
	// - queue (my theory: deltafifo)
	wq workqueue.RateLimitingInterface

	// to list pods:
	// podLister       corelisters.PodLister
	// podListerSynced cache.InformerSynced

	recorder record.EventRecorder
}

func NewController(client kubernetes.Interface, klient klientset.Interface, klusterInformer kInformer.TrackPodInformer) *Controller {
	c := &Controller{
		client:      client,
		klient:      klient,
		klusterSync: klusterInformer.Informer().HasSynced,
		klister:     klusterInformer.Lister(),
		wq:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "TrackPod"),
	}
	klusterInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.handleAdd,
			UpdateFunc: func(old, obj interface{}) {
				c.handleAdd(obj)
			},
			DeleteFunc: c.handleDel,
		},
	)

	return c
}

func (c *Controller) Run(ch chan struct{}) error {
	if ok := cache.WaitForCacheSync(ch, c.klusterSync); !ok {
		log.Println("cache was not synced")
	}
	go wait.Until(c.worker, time.Second, ch)
	<-ch
	return nil
}

func (c *Controller) worker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	item, shutdown := c.wq.Get()
	if shutdown {
		log.Println("Shutting down")
		return false
	}

	defer c.wq.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		log.Printf("error while calling Namespace Key func on cache for item %s: %s", item, err.Error())
		return false
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("error while splitting key into namespace & name: %s", err.Error())
		return false
	}

	tpod, err := c.klister.TrackPods(ns).Get(name)
	if err != nil {
		log.Printf("error %s, Getting the tpod resource from lister.", err.Error())
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
	pList, _ := c.client.CoreV1().Pods(tpod.Namespace).List(context.TODO(), listOptions)

	if err := c.syncHandler(tpod, pList); err != nil {
		log.Printf("Error while syncing the current vs desired state for TrackPod %v: %v\n", tpod.Name, err.Error())
		return false
	}
	// c.recorder.Event(tpod, corev1.EventTypeNormal, "Podcreation", "called SynHandler")

	// fmt.Println("Calling updateStatus now!!")
	// err = c.updateStatus(tpod, "creating", pList)
	// if err != nil {
	// 	log.Printf("error %s, updating status of the TrackPod %s\n", err.Error(), tpod.Name)
	// }

	fmt.Println("calling wait for pods")
	// wait for pods to be ready
	err = c.waitForPods(tpod, pList)
	if err != nil {
		log.Printf("error %s, waiting for pods to meet the expected state", err.Error())
	}

	fmt.Println("Calling update status again!!")
	err = c.updateStatus(tpod, tpod.Spec.Message, pList)
	if err != nil {
		log.Printf("error %s updating status after waiting for Pods", err.Error())
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
	pList, _ := c.client.CoreV1().Pods(tpod.Namespace).List(context.TODO(), listOptions)

	runningPods := 0
	for _, pod := range pList.Items {
		if pod.ObjectMeta.DeletionTimestamp.IsZero() && pod.Status.Phase == "Running" {
			runningPods++
		}
	}
	return runningPods
}

// If the pod doesn't switch to a running state within 10 minutes, shall report the error.
func (c *Controller) waitForPods(tpod *v1.TrackPod, pList *corev1.PodList) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		runningPods := c.totalRunningPods(tpod)
		fmt.Println("Inside waitforPods ???? totalrunningPods >>>> ", runningPods)

		if runningPods == tpod.Spec.Count {
			return true, nil
		}
		return false, nil
	})
}

// Updates the status section of TrackPod
func (c *Controller) updateStatus(tpod *v1.TrackPod, progress string, pList *corev1.PodList) error {
	t, err := c.klient.AjV1().TrackPods(tpod.Namespace).Get(context.Background(), tpod.Name, metav1.GetOptions{})
	trunningPods := c.totalRunningPods(tpod)
	if err != nil {
		return err
	}

	t.Status.Count = trunningPods
	t.Status.Message = progress
	fmt.Println("Inside updatestatus >>>>>>>>>>> ", t.Status.Message)
	_, err = c.klient.AjV1().TrackPods(tpod.Namespace).UpdateStatus(context.Background(), t, metav1.UpdateOptions{})

	return err
}

// syncHandler monitors the current state & if current != desired,
// tries to meet the desired state.
func (c *Controller) syncHandler(tpod *v1.TrackPod, pList *corev1.PodList) error {
	var podCreate, podDelete bool
	iterate := tpod.Spec.Count
	deleteIterate := 0
	runningPods := c.totalRunningPods(tpod)
	fmt.Println("Inside syncHandler >>>>>>>>>>>>>>>>>>>>> runningPods ----> ", runningPods)
	fmt.Println("======================> tpod.Count ::: ", tpod.Spec.Count)

	if runningPods < tpod.Spec.Count || tpod.Spec.Message != tpod.Status.Message {
		if tpod.Spec.Message != tpod.Status.Message {
			podDelete = true
			podCreate = true
			iterate = tpod.Spec.Count
			deleteIterate = tpod.Spec.Count
		} else {
			log.Printf("detected mismatch of replica count for CR %v!!!! expected: %v & have: %v\n\n\n", tpod.Name, tpod.Spec.Count, runningPods)
			podCreate = true
			iterate = tpod.Spec.Count - runningPods
		}
	} else if runningPods > tpod.Spec.Count {
		deleteIterate = runningPods - tpod.Spec.Count
		log.Printf("Deleting %v extra pods\n", deleteIterate)
		podDelete = true
	}

	// Delete extra pod
	// TODO: Detect the manually created pod, and delete that specific pod.
	if podDelete {
		for i := 0; i < deleteIterate; i++ {
			err := c.client.CoreV1().Pods(tpod.Namespace).Delete(context.TODO(), pList.Items[i].Name, metav1.DeleteOptions{})
			if err != nil {
				log.Printf("Pod deletion failed for CR %v\n", tpod.Name)
				return err
			}
			fmt.Println()
		}
	}

	// Creates pod
	if podCreate {
		// i := 0
		for i := 0; i < iterate; i++ {
			nPod, err := c.client.CoreV1().Pods(tpod.Namespace).Create(context.TODO(), newPod(tpod), metav1.CreateOptions{})
			if err != nil {
				if errors.IsAlreadyExists(err) {
					// retry
					iterate++
				} else {
					return err
				}
			}
			if nPod.Name != "" {
				log.Printf("Pod %v created successfully!\n", nPod.Name)
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
						"while true; do echo '$(MESSAGE)'; sleep 10; done",
					},
				},
			},
		},
	}
}

func (c *Controller) handleAdd(obj interface{}) {
	log.Println("handleAdd is here!!!")
	c.wq.Add(obj)
}

func (c *Controller) handleDel(obj interface{}) {
	log.Println("handleDel is here!!")
	c.wq.Done(obj)
}
