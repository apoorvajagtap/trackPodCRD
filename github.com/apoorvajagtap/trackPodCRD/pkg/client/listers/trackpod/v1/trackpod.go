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

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/apoorvajagtap/trackPodCRD/pkg/apis/trackpod/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TrackPodLister helps list TrackPods.
// All objects returned here must be treated as read-only.
type TrackPodLister interface {
	// List lists all TrackPods in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.TrackPod, err error)
	// TrackPods returns an object that can list and get TrackPods.
	TrackPods(namespace string) TrackPodNamespaceLister
	TrackPodListerExpansion
}

// trackPodLister implements the TrackPodLister interface.
type trackPodLister struct {
	indexer cache.Indexer
}

// NewTrackPodLister returns a new TrackPodLister.
func NewTrackPodLister(indexer cache.Indexer) TrackPodLister {
	return &trackPodLister{indexer: indexer}
}

// List lists all TrackPods in the indexer.
func (s *trackPodLister) List(selector labels.Selector) (ret []*v1.TrackPod, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.TrackPod))
	})
	return ret, err
}

// TrackPods returns an object that can list and get TrackPods.
func (s *trackPodLister) TrackPods(namespace string) TrackPodNamespaceLister {
	return trackPodNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TrackPodNamespaceLister helps list and get TrackPods.
// All objects returned here must be treated as read-only.
type TrackPodNamespaceLister interface {
	// List lists all TrackPods in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.TrackPod, err error)
	// Get retrieves the TrackPod from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.TrackPod, error)
	TrackPodNamespaceListerExpansion
}

// trackPodNamespaceLister implements the TrackPodNamespaceLister
// interface.
type trackPodNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all TrackPods in the indexer for a given namespace.
func (s trackPodNamespaceLister) List(selector labels.Selector) (ret []*v1.TrackPod, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.TrackPod))
	})
	return ret, err
}

// Get retrieves the TrackPod from the indexer for a given namespace and name.
func (s trackPodNamespaceLister) Get(name string) (*v1.TrackPod, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("trackpod"), name)
	}
	return obj.(*v1.TrackPod), nil
}
