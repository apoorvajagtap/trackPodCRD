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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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

	if err := c.syncHandler(tpod); err != nil {
		log.Printf("Error while syncing the current vs desired state for TrackPod %v: %v\n", tpod.Name, err.Error())
		return false
	}

	return true
}

// syncHandler monitors the current state & if current != desired,
// tries to meet the desired state.
func (c *Controller) syncHandler(tpod *v1.TrackPod) error {
	var podCreate, podDelete bool
	iterate := tpod.Spec.Count

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

	if len(pList.Items) < tpod.Spec.Count {
		log.Printf("detected mismatch of replica count for CR %v!!!! expected: %v & have: %v\n\n\n", tpod.Name, tpod.Spec.Count, len(pList.Items))
		podCreate = true
		iterate = tpod.Spec.Count - len(pList.Items)
	} else if len(pList.Items) > tpod.Spec.Count {
		log.Println("Deleting one of the extra pods")
		podDelete = true
	}

	// Creates pod
	if podCreate {
		for i := 0; i < iterate; i++ {
			nPod, err := c.client.CoreV1().Pods(tpod.Namespace).Create(context.TODO(), newPod(tpod), metav1.CreateOptions{})
			if err != nil {
				return err
			}
			log.Printf("Pod %v created successfully!\n", nPod.Name)
		}
	}

	// Delete extra pod
	// TODO: Detect the manually created pod, and delete that specific pod.
	if podDelete {
		err := c.client.CoreV1().Pods(tpod.Namespace).Delete(context.TODO(), pList.Items[0].Name, metav1.DeleteOptions{})
		if err != nil {
			log.Printf("Pod deletion failed for CR %v\n", tpod.Name)
		}
		return err
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
			Name:      fmt.Sprintf(tpod.Name + "-" + strconv.Itoa(rand.Intn(10000))),
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
