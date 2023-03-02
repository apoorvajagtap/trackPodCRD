package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*Adding following tag, because we want to generate ClientSet for following type*/
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TaskRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskRunSpec   `json:"spec,omitempty"`
	Status TaskRunStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TaskRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TaskRun `json:"items"`
}

type TaskRunSpec struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type TaskRunStatus struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}
