package v1

import (
	"github.com/apoorvajagtap/trackPodCRD/pkg/apis/trackpod"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   trackpod.GroupName,
	Version: "v1",
}

var (
	SchemeBuilder runtime.SchemeBuilder
	AddToScheme   = SchemeBuilder.AddToScheme
)

// as soon as package is loaded, it should use the 'TrackPod' type (struct)
func init() {
	// function will register our specified type to the scheme.
	SchemeBuilder.Register(addKnownTypes)
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&TrackPod{},
		&TrackPodList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}

// for Resource func reference in listers
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
