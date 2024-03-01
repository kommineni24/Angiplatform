/*
Copyright 2024 Ramakrishna Kommineni.

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

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	myapigroupv1alpha1 "github.com/kommineni24/k8appcontroller/api/v1alpha1"
)

// MyAppResourceReconciler reconciles a MyAppResource object
type MyAppResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=my.api.group.rama.angi.platform,resources=myappresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.api.group.rama.angi.platform,resources=myappresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.api.group.rama.angi.platform,resources=myappresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MyAppResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("myappresource", req.NamespacedName)

	// Fetch the MyAppResource instance
	myAppResource := &myapigroupv1alpha1.MyAppResource{}
	if err := r.Get(ctx, req.NamespacedName, myAppResource); err != nil {
		log.Error(err, "Failed to fetch MyAppResource")
		return ctrl.Result{}, err
	}

	// Reconciliation logic
	replicaCount := myAppResource.Spec.ReplicaCount
	image := myAppResource.Spec.Image
	resources := myAppResource.Spec.Resources
	ui := myAppResource.Spec.UI
	redisEnabled := myAppResource.Spec.Redis.Enabled
	redisReplicaCount := int32(1) // Default to 1 replica

	// Check if replica count for Redis is specified
	if myAppResource.Spec.Redis.ReplicaCount != nil {
		redisReplicaCount = *myAppResource.Spec.Redis.ReplicaCount
	}

	// Deploy main application pods
	if replicaCount > 0 {
		// Create or delete pods based on replica count
		podList := &corev1.PodList{}
		if err := r.List(ctx, podList, client.InNamespace(req.Namespace), client.MatchingLabels{"app": req.Name}); err != nil {
			log.Error(err, "Failed to list pods")
			return ctrl.Result{}, err
		}

		for _, pod := range podList.Items {
			// Update pod's image and resources if they differ from the spec
			var updated bool
			for i, container := range pod.Spec.Containers {
				if container.Name == "app-container" {
					if container.Image != fmt.Sprintf("%s:%s", image.Repository, image.Tag) {
						pod.Spec.Containers[i].Image = fmt.Sprintf("%s:%s", image.Repository, image.Tag)
						updated = true
					}
					if cpuReq, ok := container.Resources.Requests[corev1.ResourceCPU]; ok && cpuReq.String() != resources.CPURequest {
						pod.Spec.Containers[i].Resources.Requests[corev1.ResourceCPU] = resource.MustParse(resources.CPURequest)
						updated = true
					}
					if memLimit, ok := container.Resources.Limits[corev1.ResourceMemory]; ok && memLimit.String() != resources.MemoryLimit {
						pod.Spec.Containers[i].Resources.Limits[corev1.ResourceMemory] = resource.MustParse(resources.MemoryLimit)
						updated = true
					}
					break // No need to continue iterating once we've found the app container
				}
			}
			// If any updates were made, update the pod
			if updated {
				if err := r.Update(ctx, &pod); err != nil {
					log.Error(err, "Failed to update pod", "Namespace", pod.Namespace, "Name", pod.Name)
					return ctrl.Result{}, err
				}
				log.Info("Updated pod", "Namespace", pod.Namespace, "Name", pod.Name)
				// Add logic to handle the updated pod if needed
			}
		}

		currentReplicaCount := int32(len(podList.Items))

		if currentReplicaCount < replicaCount {
			for i := currentReplicaCount; i < replicaCount; i++ {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", req.Name, i),
						Namespace: req.Namespace,
						Labels: map[string]string{
							"app":   req.Name,
							"color": ui.Color,
						},
						Annotations: map[string]string{
							"message": ui.Message,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app-container",
								Image: fmt.Sprintf("%s:%s", image.Repository, image.Tag),
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse(resources.CPURequest),
										corev1.ResourceMemory: resource.MustParse(resources.MemoryLimit),
									},
								},
							},
						},
					},
				}
				if err := ctrl.SetControllerReference(myAppResource, pod, r.Scheme); err != nil {
					log.Error(err, "Failed to set controller reference")
					return ctrl.Result{}, err
				}
				if err := r.Create(ctx, pod); err != nil {
					log.Error(err, "Failed to create pod", "Namespace", pod.Namespace, "Name", pod.Name)
					return ctrl.Result{}, err
				}
			}
		} else if currentReplicaCount > replicaCount {
			for i := currentReplicaCount - 1; i >= replicaCount; i-- {
				// Get the ObjectKey of the pod for deletion
				pod := podList.Items[i]
				podKey := client.ObjectKey{Namespace: pod.Namespace, Name: pod.Name}
				if err := r.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: req.Namespace, Name: pod.Name}}); err != nil {
					log.Error(err, "Failed to delete pod", "Namespace", pod.Namespace, "Name", pod.Name)
					return ctrl.Result{}, err
				}
				// Ensure that the pod is deleted by waiting until it's no longer found
				if err := r.waitForDeletion(ctx, podKey); err != nil {
					log.Error(err, "Failed to wait for pod deletion", "Namespace", pod.Namespace, "Name", pod.Name)
					// You can choose to return an error here if desired
				}
			}
		}
	}

	// Deploy Redis instance if enabled
	if redisEnabled {
		// Define Redis deployment
		redisDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-redis", myAppResource.Name),
				Namespace: myAppResource.Namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &redisReplicaCount,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": myAppResource.Name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": myAppResource.Name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "redis",
								Image: "redis:latest",
							},
						},
					},
				},
			},
		}

		// Set MyAppResource instance as the owner and controller
		if err := ctrl.SetControllerReference(myAppResource, redisDeployment, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for Redis deployment")
			return ctrl.Result{}, err
		}

		// Check if the Redis deployment exists
		found := &appsv1.Deployment{}
		err := r.Get(ctx, client.ObjectKey{Namespace: redisDeployment.Namespace, Name: redisDeployment.Name}, found)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating Redis deployment", "Namespace", redisDeployment.Namespace, "Name", redisDeployment.Name)
			err = r.Create(ctx, redisDeployment)
			if err != nil {
				log.Error(err, "Failed to create Redis deployment", "Namespace", redisDeployment.Namespace, "Name", redisDeployment.Name)
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get Redis deployment")
			return ctrl.Result{}, err
		}

		// Ensure the deployment size is the same as the spec
		// size := redisReplicaCount
		// if *found.Spec.Replicas != size {
		// 	found.Spec.Replicas = &size
		// 	err = r.Update(ctx, found)
		// 	if err != nil {
		// 		log.Error(err, "Failed to update Redis deployment", "Namespace", found.Namespace, "Name", found.Name)
		// 		return ctrl.Result{}, err
		// 	}
		// }

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myapigroupv1alpha1.MyAppResource{}).
		Complete(r)
}

// Function to wait for pod deletion
func (r *MyAppResourceReconciler) waitForDeletion(ctx context.Context, podKey client.ObjectKey) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * 5):
			// Check if the pod still exists
			pod := &corev1.Pod{}
			err := r.Get(ctx, podKey, pod)
			if err != nil && errors.IsNotFound(err) {
				// Pod is deleted
				return nil
			} else if err != nil {
				// Error occurred while fetching the pod
				return err
			}
		}
	}
}
