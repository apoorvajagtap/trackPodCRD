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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1alpha1 "github.com/apoorvajagtap/trackPodCRD/pkg/apis/pipeline/v1alpha1"
	pipelinev1alpha1 "github.com/apoorvajagtap/trackPodCRD/pkg/client/applyconfiguration/pipeline/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePipelineRuns implements PipelineRunInterface
type FakePipelineRuns struct {
	Fake *FakeAjV1alpha1
	ns   string
}

var pipelinerunsResource = v1alpha1.SchemeGroupVersion.WithResource("pipelineruns")

var pipelinerunsKind = v1alpha1.SchemeGroupVersion.WithKind("PipelineRun")

// Get takes name of the pipelineRun, and returns the corresponding pipelineRun object, and an error if there is any.
func (c *FakePipelineRuns) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.PipelineRun, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(pipelinerunsResource, c.ns, name), &v1alpha1.PipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PipelineRun), err
}

// List takes label and field selectors, and returns the list of PipelineRuns that match those selectors.
func (c *FakePipelineRuns) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.PipelineRunList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(pipelinerunsResource, pipelinerunsKind, c.ns, opts), &v1alpha1.PipelineRunList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.PipelineRunList{ListMeta: obj.(*v1alpha1.PipelineRunList).ListMeta}
	for _, item := range obj.(*v1alpha1.PipelineRunList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested pipelineRuns.
func (c *FakePipelineRuns) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(pipelinerunsResource, c.ns, opts))

}

// Create takes the representation of a pipelineRun and creates it.  Returns the server's representation of the pipelineRun, and an error, if there is any.
func (c *FakePipelineRuns) Create(ctx context.Context, pipelineRun *v1alpha1.PipelineRun, opts v1.CreateOptions) (result *v1alpha1.PipelineRun, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(pipelinerunsResource, c.ns, pipelineRun), &v1alpha1.PipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PipelineRun), err
}

// Update takes the representation of a pipelineRun and updates it. Returns the server's representation of the pipelineRun, and an error, if there is any.
func (c *FakePipelineRuns) Update(ctx context.Context, pipelineRun *v1alpha1.PipelineRun, opts v1.UpdateOptions) (result *v1alpha1.PipelineRun, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(pipelinerunsResource, c.ns, pipelineRun), &v1alpha1.PipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PipelineRun), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePipelineRuns) UpdateStatus(ctx context.Context, pipelineRun *v1alpha1.PipelineRun, opts v1.UpdateOptions) (*v1alpha1.PipelineRun, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(pipelinerunsResource, "status", c.ns, pipelineRun), &v1alpha1.PipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PipelineRun), err
}

// Delete takes name of the pipelineRun and deletes it. Returns an error if one occurs.
func (c *FakePipelineRuns) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(pipelinerunsResource, c.ns, name, opts), &v1alpha1.PipelineRun{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePipelineRuns) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(pipelinerunsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.PipelineRunList{})
	return err
}

// Patch applies the patch and returns the patched pipelineRun.
func (c *FakePipelineRuns) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.PipelineRun, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(pipelinerunsResource, c.ns, name, pt, data, subresources...), &v1alpha1.PipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PipelineRun), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied pipelineRun.
func (c *FakePipelineRuns) Apply(ctx context.Context, pipelineRun *pipelinev1alpha1.PipelineRunApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.PipelineRun, err error) {
	if pipelineRun == nil {
		return nil, fmt.Errorf("pipelineRun provided to Apply must not be nil")
	}
	data, err := json.Marshal(pipelineRun)
	if err != nil {
		return nil, err
	}
	name := pipelineRun.Name
	if name == nil {
		return nil, fmt.Errorf("pipelineRun.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(pipelinerunsResource, c.ns, *name, types.ApplyPatchType, data), &v1alpha1.PipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PipelineRun), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakePipelineRuns) ApplyStatus(ctx context.Context, pipelineRun *pipelinev1alpha1.PipelineRunApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.PipelineRun, err error) {
	if pipelineRun == nil {
		return nil, fmt.Errorf("pipelineRun provided to Apply must not be nil")
	}
	data, err := json.Marshal(pipelineRun)
	if err != nil {
		return nil, err
	}
	name := pipelineRun.Name
	if name == nil {
		return nil, fmt.Errorf("pipelineRun.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(pipelinerunsResource, c.ns, *name, types.ApplyPatchType, data, "status"), &v1alpha1.PipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PipelineRun), err
}