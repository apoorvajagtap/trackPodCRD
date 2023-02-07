package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TrackPodSpec struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type TrackPodStatus struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

/*Adding following tag, because we want to generate ClientSet for following type*/
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TrackPod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrackPodSpec   `json:"spec,omitempty"`
	Status TrackPodStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TrackPodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TrackPod `json:"items"`
}
