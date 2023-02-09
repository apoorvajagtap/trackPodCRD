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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
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
	podLister       corelisters.PodLister
	podListerSynced cache.InformerSynced
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
	// Check if current count of pods == tpod.spec.count or not
	// labelSelector := metav1.LabelSelector{
	// 	MatchLabels: map[string]string{
	// 		"controller": tpod.Name,
	// 	},
	// }
	// listOptions := metav1.ListOptions{
	// 	LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	// }
	// podList, _ := c.client.CoreV1().Pods(tpod.Namespace).List(context.TODO(), listOptions)

	// podList, _ := c.client.CoreV1().Pods(tpod.Namespace).List(context.TODO(), metav1.ListOptions{
	// 	FieldSelector: "metadata.ownerReferences[0].kind=TrackPod",
	// })

	// Creates pod
	for i := 0; i < tpod.Spec.Count; i++ {
		nPod, err := c.client.CoreV1().Pods(tpod.Namespace).Create(context.TODO(), newPod(tpod), metav1.CreateOptions{})
		if err != nil {
			return err
		}
		fmt.Printf("Pod %v created successfully!\n", nPod.Name)
	}
	return nil
}

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
	// Should create pods using the spec details from the CR.
	log.Println("handleAdd is here!!!")
	c.wq.Add(obj)
	// if err := c.createPod(obj); err != nil {
	// 	log.Printf("Error while creating the pod: %s\n", err.Error())
	// } else {
	// 	log.Printf("Pod created successfully\n")
	// }
}

func (c *Controller) handleDel(obj interface{}) {
	log.Println("handleDel is here!!")
	c.wq.Done(obj)
}
