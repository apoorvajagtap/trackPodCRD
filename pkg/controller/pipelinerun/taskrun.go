package pipelinerun

import (
	"context"
	"fmt"
	"time"

	"github.com/apoorvajagtap/trackPodCRD/pkg/apis/pipeline/v1alpha1"
	"github.com/kanisterio/kanister/pkg/poll"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

func (c *Controller) createPodTask(prun *v1alpha1.PipelineRun, trun *v1alpha1.TaskRun) error {
	// var podCreate, podDelete bool
	iterate := trun.Spec.Count
	// Creates pod
	for i := 0; i < iterate; i++ {
		nPod, err := c.kubeClient.CoreV1().Pods(trun.Namespace).Create(context.TODO(), newPod(trun), metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("Pod creation failed for CR %v\n", trun.Name)
			return err
		}
		if nPod.Name != "" {
			klog.Infof("Pod %v created successfully!\n", nPod.Name)
		}
	}

	return nil
}

// Creates the new pod with the specified template
func newPod(trun *v1alpha1.TaskRun) *corev1.Pod {
	labels := map[string]string{
		"controller": trun.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:       labels,
			GenerateName: fmt.Sprintf("%s-", trun.Name),
			Namespace:    trun.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(trun, v1alpha1.SchemeGroupVersion.WithKind("TaskRun")),
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: "Never",
			Containers: []corev1.Container{
				{
					Name:  "static-nginx",
					Image: "nginx:latest",
					Env: []corev1.EnvVar{
						{
							Name:  "MESSAGE",
							Value: trun.Spec.Message,
						},
					},
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						"echo '$(MESSAGE)'",
					},
				},
			},
		},
	}
}

// total number of 'Completed' pods
func (c *Controller) totalCompletedPods(trun *v1alpha1.TaskRun) int {
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"controller": trun.Name,
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	// TODO: Prefer using podLister to reduce the call to K8s API.
	pList, _ := c.kubeClient.CoreV1().Pods(trun.Namespace).List(context.TODO(), listOptions)

	completedPods := 0
	for _, pod := range pList.Items {
		if pod.ObjectMeta.DeletionTimestamp.IsZero() && pod.Status.Phase == "Succeeded" {
			completedPods++
		}
	}
	return completedPods
}

// If the pod doesn't switch to a running state within 10 minutes, shall report.
func (c *Controller) waitForPods(trun *v1alpha1.TaskRun) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		completedPods := c.totalCompletedPods(trun)
		fmt.Println("Inside waitforPods ???? totalcompletedpods >>>> ", completedPods)

		if completedPods == trun.Spec.Count {
			return true, nil
		}
		return false, nil
	})
}

func (c *Controller) updateTrunStatus(trun *v1alpha1.TaskRun) error {
	t, err := c.prunClient.AjV1alpha1().TaskRuns(trun.Namespace).Get(context.Background(), trun.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	fmt.Println("about to update trun status >>>> ", trun)
	klog.Infof("Insider updatetrunstatus: %v ,,, %v", c.totalCompletedPods(trun), t.Spec.Message, t.Spec.Count)

	t.Status.Count = c.totalCompletedPods(trun)
	t.Status.Message = t.Spec.Message
	_, err = c.prunClient.AjV1alpha1().TaskRuns(trun.Namespace).UpdateStatus(context.Background(), t, metav1.UpdateOptions{})

	return err
}
