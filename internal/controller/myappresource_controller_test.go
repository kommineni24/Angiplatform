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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	myapigroupv1alpha1 "github.com/kommineni24/k8appcontroller/api/v1alpha1"
)

var _ = Describe("MyAppResource Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		myAppResource := &myapigroupv1alpha1.MyAppResource{}
		var reconciler *MyAppResourceReconciler

		BeforeEach(func() {
			By("creating the custom resource for the Kind MyAppResource")
			err := k8sClient.Get(ctx, typeNamespacedName, myAppResource)
			if err != nil && errors.IsNotFound(err) {
				resource := &myapigroupv1alpha1.MyAppResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &myapigroupv1alpha1.MyAppResource{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance MyAppResource")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &MyAppResourceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})

		It("should handle error when custom resource doesn't exist", func() {
			// Set up the environment by deleting the custom resource
			// that was created in BeforeEach
			resource := &myapigroupv1alpha1.MyAppResource{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			// Reconcile the resource and expect an error
			controllerReconciler := &MyAppResourceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
		})

		// Test case for creating pods
		It("should create pods based on replica count", func() {
			// Setup
			ctx := context.Background()
			resourceName := "test-resource"
			typeNamespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"} // Modify namespace as needed
			var replicaCount = 4
			// Reconcile the resource
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Verify that pods are created
			podList := &corev1.PodList{}
			err = k8sClient.List(ctx, podList, client.InNamespace(typeNamespacedName.Namespace), client.MatchingLabels{"app": resourceName})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(podList.Items)).To(Equal(int(replicaCount))) // Assuming replicaCount is a predefined value
		})

		// Test case for updating pods
		It("should update pods when custom resource is updated", func() {
			// Setup
			ctx := context.Background()
			resourceName := "test-resource"
			typeNamespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"} // Modify namespace as needed

			// Reconcile the resource
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Modify custom resource spec
			myAppResource.Spec.Image.Repository = "new-repo" // Modify the image repository
			myAppResource.Spec.Resources.CPURequest = "100m" // Modify CPU request

			// Update the custom resource
			err = k8sClient.Update(ctx, myAppResource)
			Expect(err).NotTo(HaveOccurred())

			// Reconcile the resource again to trigger the update
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Verify that pods are updated
			podList := &corev1.PodList{}
			err = k8sClient.List(ctx, podList, client.InNamespace(typeNamespacedName.Namespace), client.MatchingLabels{"app": resourceName})
			Expect(err).NotTo(HaveOccurred())
			for _, pod := range podList.Items {
				for _, container := range pod.Spec.Containers {
					if container.Name == "app-container" {
						Expect(container.Image).To(Equal("new-repo:latest"))
						cpuRequest, cpuFound := container.Resources.Requests[corev1.ResourceCPU]
						Expect(cpuFound).To(BeTrue())                 // Check if CPU request is defined
						Expect(cpuRequest.String()).To(Equal("100m")) // Use CPU request if defined
						// You can add more assertions for other fields as needed
						break
					}
				}
			}
		})

		// Test case for deleting pods
		It("should delete excess pods when replica count is decreased", func() {
			// Setup
			ctx := context.Background()
			resourceName := "test-resource"
			typeNamespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"} // Modify namespace as needed

			// Reconcile the resource
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Modify custom resource spec to decrease replica count
			myAppResource.Spec.ReplicaCount = 1 // Set replica count to 1
			err = k8sClient.Update(ctx, myAppResource)
			Expect(err).NotTo(HaveOccurred())

			// Reconcile the resource again to trigger deletion of excess pods
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Verify that excess pods are deleted
			podList := &corev1.PodList{}
			err = k8sClient.List(ctx, podList, client.InNamespace(typeNamespacedName.Namespace), client.MatchingLabels{"app": resourceName})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(podList.Items)).To(Equal(1)) // Expect only one pod to remain after deletion
		})

		// Test case for deploying Redis
		It("should deploy Redis when enabled in custom resource", func() {
			// Setup
			ctx := context.Background()
			resourceName := "test-resource"
			typeNamespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"} // Modify namespace as needed

			// Enable Redis in custom resource
			myAppResource.Spec.Redis.Enabled = true
			err := k8sClient.Update(ctx, myAppResource)
			Expect(err).NotTo(HaveOccurred())

			// Reconcile the resource to trigger Redis deployment
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Verify that Redis deployment is created
			redisDeployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: fmt.Sprintf("%s-redis", resourceName)}, redisDeployment)
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
