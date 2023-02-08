package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	klient "github.com/apoorvajagtap/trackPodCRD/pkg/client/clientset/versioned"
	kInfFac "github.com/apoorvajagtap/trackPodCRD/pkg/client/informers/externalversions"
	"github.com/apoorvajagtap/trackPodCRD/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf("Building config from flags failed, %s, trying to build inclusterconfig", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("error %s building inclusterconfig", err.Error())
		}
	}

	// creating the clientset
	klientset, err := klient.NewForConfig(config)
	if err != nil {
		log.Printf("getting klient set %s\n", err.Error())
	}
	fmt.Println(klientset)

	tpods, err := klientset.AjV1().TrackPods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Printf("error while listing trackPods %s\n", err.Error())
	}
	fmt.Println(tpods)
	fmt.Printf("total trackPod sets: %d\n", len(tpods.Items))

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("getting std client %s\n", err.Error())
	}

	infoFact := kInfFac.NewSharedInformerFactory(klientset, 20*time.Minute)
	ch := make(chan struct{})
	c := controller.NewController(client, klientset, infoFact.Aj().V1().TrackPods())

	infoFact.Start(ch)
	if err := c.Run(ch); err != nil {
		log.Printf("error running controller %s\n", err)
	}
}

// if err := createPod(); err != nil {
// 	log.Printf("Error while creating the pod: %s\n", err.Error())
// } else {
// 	log.Printf("Pod created successfully\n")
// }
