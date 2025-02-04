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
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"strconv"
	"time"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	pb "github.com/opplieam/bb-dist-noti/protogen/notification_v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	noderolev1 "github.com/opplieam/bb-dist-noti-operator/api/v1"
)

// LeaderStatusCheckerReconciler reconciles a LeaderStatusChecker object
type LeaderStatusCheckerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=node-role.bb-noti.io,resources=leaderstatuscheckers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=node-role.bb-noti.io,resources=leaderstatuscheckers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=node-role.bb-noti.io,resources=leaderstatuscheckers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LeaderStatusChecker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *LeaderStatusCheckerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	var leaderStatusChecker noderolev1.LeaderStatusChecker
	if err := r.Get(ctx, req.NamespacedName, &leaderStatusChecker); err != nil {
		if errors.IsNotFound(err) {
			l.Info("LeaderStatusChecker resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		l.Error(err, "Failed to get LeaderStatusChecker.")
		return ctrl.Result{}, err
	}

	intervalSeconds := leaderStatusChecker.Spec.IntervalSeconds
	statefulSetName := leaderStatusChecker.Spec.StatefulSetName
	namespace := leaderStatusChecker.Spec.Namespace
	grpcPort := leaderStatusChecker.Spec.RPCPort
	localDev := leaderStatusChecker.Spec.LocalDev

	l.Info("Reconciling LeaderStatusChecker", "intervalSeconds",
		intervalSeconds, "statefulSetName", statefulSetName, "namespace", namespace)

	// Fetch the StatefulSet
	var statefulSet appsv1.StatefulSet
	err := r.Get(ctx, types.NamespacedName{Name: statefulSetName, Namespace: namespace}, &statefulSet)
	if err != nil {
		if errors.IsNotFound(err) {
			l.Info("StatefulSet not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		l.Error(err, "Failed to get StatefulSet.")
		return ctrl.Result{}, err
	}

	// List pods owned by this StatefulSet
	var podList corev1.PodList
	err = r.List(ctx, &podList, client.InNamespace(namespace),
		client.MatchingLabels(statefulSet.Spec.Selector.MatchLabels))
	if err != nil {
		l.Error(err, "Failed to list Pods.")
		return ctrl.Result{}, err
	}

	// checkLeaderStatus use Full Qualified Domain Name (FQDN) to connect to the gRPC server
	// It only works when operator run on the same cluster
	// Quick hack for fast development is to skip external gRPC called
	// It randomly picks a leader pod and sets the node-role label to "leader"
	// If the pod is not leader, it sets the node-role label to "follower"
	// TODO: Add support for external cluster
	pickLeader := rand.IntN(len(podList.Items))

	for i, pod := range podList.Items {
		podName := pod.Name
		podHostname := fmt.Sprintf("%s.%s.%s.svc.cluster.local", podName, statefulSetName, namespace)
		grpcAddr := net.JoinHostPort(podHostname, strconv.Itoa(int(grpcPort)))

		var isLeader bool
		var podErr error
		if localDev {
			isLeader = i == pickLeader
			podErr = nil
		} else {
			isLeader, podErr = r.checkLeaderStatus(ctx, grpcAddr)
		}

		if podErr != nil {
			l.Error(podErr, "Failed to check leader status.")
			continue
		}

		expectedLabels := "follower"
		if isLeader {
			expectedLabels = "leader"
		}
		currentLabels := pod.GetLabels()

		if currentLabels["node-role"] != expectedLabels {
			currentLabels["node-role"] = expectedLabels
			pod.SetLabels(currentLabels)

			if err = r.Update(ctx, &pod); err != nil {
				l.Error(err, "Failed to update Pod labels.", "pod", podName)
				continue
			}

			leaderStatusChecker.Status.LastUpdated = metav1.Now()
			if err = r.Status().Update(ctx, &leaderStatusChecker); err != nil {
				l.Error(err, "Failed to update CR status", "pod", podName)
				continue
			}
		} else {
			l.V(1).Info("Pod labels are already correct.",
				"pod", podName, "role", expectedLabels)
		}

	}

	l.V(1).Info("Successfully reconciled LeaderStatusChecker.")
	return ctrl.Result{RequeueAfter: time.Duration(intervalSeconds) * time.Second}, nil
}

func (r *LeaderStatusCheckerReconciler) checkLeaderStatus(ctx context.Context, grpcAddr string) (bool, error) {
	l := log.FromContext(ctx)

	dialOps := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.NewClient(grpcAddr, dialOps...)

	if err != nil {
		return false, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			l.Error(closeErr, "failed to close gRPC client")
		}
	}()

	c := pb.NewNotificationClient(conn)
	resp, err := c.GetLeaderStatus(ctx, &emptypb.Empty{})
	if err != nil {
		l.Error(err, "failed to call GetLeaderStatus")
		return false, fmt.Errorf("failed to call GetLeaderStatus: %w", err)
	}
	return resp.GetIsLeader(), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LeaderStatusCheckerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&noderolev1.LeaderStatusChecker{}).
		Named("leaderstatuschecker").
		Complete(r)
}
