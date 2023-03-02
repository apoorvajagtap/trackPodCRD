/*essential to specify global tags for our project.
& tags help us to basically control the behavior of code generator*/

// +k8s:deepcopy-gen=package,register
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true
// +k8s:conversion-gen=github.com/apoorvajagtap/trackPodCRD/pkg/apis/pipeline
// +groupName=aj.com

package v2
