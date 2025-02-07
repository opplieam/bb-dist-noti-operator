/*
Copyright 2025.

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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	noderolev1 "github.com/opplieam/bb-dist-noti-operator/api/v1"
)

var _ = Describe("LeaderStatusChecker Controller", func() {
	const (
		resourceName      = "test-leaderstatuschecker"
		statefulSetName   = "test-statefulset"
		resourceNamespace = "default"
		grpcPort          = 8000
	)
	var (
		leaderStatusChecker *noderolev1.LeaderStatusChecker
		statefulSet         *appsv1.StatefulSet
		typeNamespacedName  = types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
	)

	BeforeEach(func() {
		leaderStatusChecker = &noderolev1.LeaderStatusChecker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: resourceNamespace,
			},
			Spec: noderolev1.LeaderStatusCheckerSpec{
				StatefulSetName: statefulSetName,
				Namespace:       resourceNamespace,
				RPCPort:         grpcPort,
				// Enable LocalDev for unit tests to skip gRPC
				LocalDev: true,
			},
		}

		statefulSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      statefulSetName,
				Namespace: resourceNamespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-app",
					},
				},
				Replicas: intPtr(3),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test-app"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "test-container",
							Image: "test-image",
						}},
					},
				},
			},
		}

	})

	AfterEach(func() {
		// Cleanup resources after each test
		err := k8sClient.Delete(ctx, leaderStatusChecker)
		if err != nil {
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred(), "Error during LeaderStatusChecker deletion (other than NotFound)")
			}
		}

		err = k8sClient.Delete(ctx, statefulSet)
		if err != nil {
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred(), "Error during StatefulSet deletion (other than NotFound)")
			}
		}
	})

	Context("Reconcile", func() {
		It("should handle resource not found", func() {
			cr := &LeaderStatusCheckerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := cr.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
		})
		It("should handle StatefulSet not found", func() {
			Expect(k8sClient.Create(ctx, leaderStatusChecker)).To(Succeed())

			cr := &LeaderStatusCheckerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := cr.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
		})
		It("should successfully reconcile and update pod labels", func() {
			Expect(k8sClient.Create(ctx, leaderStatusChecker)).To(Succeed())
			Expect(k8sClient.Create(ctx, statefulSet)).To(Succeed())

			// Create pods for the StatefulSet (simulating StatefulSet controller)
			for i := range *statefulSet.Spec.Replicas {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      statefulSetName + "-" + strconv.Itoa(int(i)),
						Namespace: resourceNamespace,
						Labels:    statefulSet.Spec.Selector.MatchLabels,
					},
					Spec: statefulSet.Spec.Template.Spec,
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				}
				Expect(k8sClient.Create(ctx, pod)).To(Succeed(), "Failed to create pod")
			}

			cr := &LeaderStatusCheckerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := cr.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred(), "Reconcile should not return an error")

			// Verify Pod labels are updated. At least one pod should be leader.
			podList := &corev1.PodList{}
			Expect(k8sClient.List(ctx,
				podList,
				controllerclient.InNamespace(resourceNamespace),
				controllerclient.MatchingLabels(statefulSet.Spec.Selector.MatchLabels),
			)).To(Succeed())

			leaderCount := 0
			followerCount := 0
			for _, pod := range podList.Items {
				labels := pod.GetLabels()
				Expect(labels).Should(HaveKey("node-role"))
				if labels["node-role"] == "leader" {
					leaderCount++
				} else if labels["node-role"] == "follower" {
					followerCount++
				} else {
					Fail("Unexpected node-role label value")
				}
			}
			Expect(leaderCount).To(Equal(1), "Expected 1 leader pod")
			// Rest should be followers
			Expect(followerCount).To(Equal(int(*statefulSet.Spec.Replicas) - 1))

			// Verify LeaderStatusChecker status is updated
			updatedLeaderStatusChecker := &noderolev1.LeaderStatusChecker{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedLeaderStatusChecker)).To(Succeed())
			Expect(updatedLeaderStatusChecker.Status.LastUpdated).NotTo(BeNil(),
				"LastUpdated should not be nil")
			Expect(updatedLeaderStatusChecker.Status.LeaderNode).NotTo(BeEmpty(),
				"LeaderNode should not be empty")

		})
	})

})

func intPtr(val int32) *int32 {
	return &val
}
