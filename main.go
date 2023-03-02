package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"

	klient "github.com/apoorvajagtap/trackPodCRD/pkg/client/clientset/versioned"
	kInfFac "github.com/apoorvajagtap/trackPodCRD/pkg/client/informers/externalversions"
	"github.com/apoorvajagtap/trackPodCRD/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/code-generator"
)

func main() {
	// find the kubeconfig file
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String(
			"kubeconfig",
			filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// Building config from flags might fail inside the pod,
	// hence adding the code for usage of in-clusterconfig.
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		klog.Errorf("Building config from flags failed, %s, trying to build inclusterconfig", err.Error())
		// uses serviceAccount mounted inside the pod.
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Errorf("error %s building inclusterconfig", err.Error())
		}
	}

	// creating the clientset
	klientset, err := klient.NewForConfig(config)
	if err != nil {
		klog.Errorf("getting klient set %s\n", err.Error())
	}
	// fmt.Println(klientset)

	// Listing the existing trackpods.
	tpods, err := klientset.AjV1().TrackPods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("error while listing trackPods %s\n", err.Error())
	}
	fmt.Println(tpods)
	// fmt.Printf("total trackPod sets: %d\n", len(tpods.Items))

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("getting std client %s\n", err.Error())
	}

	infoFact := kInfFac.NewSharedInformerFactory(klientset, 20*time.Minute)
	ch := make(chan struct{})
	c := controller.NewController(client, klientset, infoFact.Aj().V1().TrackPods())

	infoFact.Start(ch)
	if err := c.Run(ch); err != nil {
		klog.Errorf("error running controller %s\n", err)
	}
}
