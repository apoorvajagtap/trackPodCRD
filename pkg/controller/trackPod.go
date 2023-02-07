package controller

import (
	"log"
	"time"

	klientset "github.com/apoorvajagtap/trackPodCRD/pkg/client/clientset/versioned"
	kInformer "github.com/apoorvajagtap/trackPodCRD/pkg/client/informers/externalversions/aj.com/v1"
	klientLister "github.com/apoorvajagtap/trackPodCRD/pkg/client/listers/aj.com/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	// things required for controller:
	// - clientset for custom resource
	klient klientset.Interface
	// - resource (informer) cache has synced
	klusterSync cache.InformerSynced
	// - interface provided by informer
	klister klientLister.TrackPodLister
	// - queue (my theory: deltafifo)
	wq workqueue.RateLimitingInterface
}

func NewController(klient klientset.Interface, klusterInformer kInformer.TrackPodInformer) *Controller {
	c := &Controller{
		klient:      klient,
		klusterSync: klusterInformer.Informer().HasSynced,
		klister:     klusterInformer.Lister(),
		wq:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "TrackPod"),
	}
	klusterInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
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
	log.Printf("TrackPods spec that we have is %+v & name of tpod is: %s\n", tpod.Spec, tpod.Name)

	return true
}

func (c *Controller) handleAdd(obj interface{}) {
	log.Println("handleAdd is here!!!")
	c.wq.Add(obj)
}

func (c *Controller) handleDel(obj interface{}) {
	log.Println("handleDel is here!!")
	c.wq.Done(obj)
}
