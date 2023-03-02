/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	ajcomv1 "github.com/apoorvajagtap/trackPodCRD/pkg/apis/trackpod/v1"
	versioned "github.com/apoorvajagtap/trackPodCRD/pkg/client/clientset/versioned"
	internalinterfaces "github.com/apoorvajagtap/trackPodCRD/pkg/client/informers/externalversions/internalinterfaces"
	v1 "github.com/apoorvajagtap/trackPodCRD/pkg/client/listers/aj.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// TrackPodInformer provides access to a shared informer and lister for
// TrackPods.
type TrackPodInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.TrackPodLister
}

type trackPodInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewTrackPodInformer constructs a new informer for TrackPod type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewTrackPodInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredTrackPodInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredTrackPodInformer constructs a new informer for TrackPod type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredTrackPodInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AjV1().TrackPods(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AjV1().TrackPods(namespace).Watch(context.TODO(), options)
			},
		},
		&ajcomv1.TrackPod{},
		resyncPeriod,
		indexers,
	)
}

func (f *trackPodInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredTrackPodInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *trackPodInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&ajcomv1.TrackPod{}, f.defaultInformer)
}

func (f *trackPodInformer) Lister() v1.TrackPodLister {
	return v1.NewTrackPodLister(f.Informer().GetIndexer())
}
