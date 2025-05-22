/*
Copyright 2024 xiloss.

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
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	log "github.com/sirupsen/logrus"
	appsv1alpha1 "github.com/xiloss/kubernats/api/v1alpha1"
)

// JetStreamReconciler reconciles a JetStream object
type JetStreamReconciler struct {
	KubeClient client.Client
	NATSClient *ClientImpl
	Scheme     *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=jetstreams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=jetstreams/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=jetstreams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the JetStream object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *JetStreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.WithFields(log.Fields{
		"namespace": req.Namespace,
		"name":      req.Name,
	})

	log.Info("starting reconciliation")

	// Fetch the NATSJetStream instance
	var jetStream appsv1alpha1.JetStream
	if err := r.KubeClient.Get(ctx, req.NamespacedName, &jetStream); err != nil {
		logger.WithError(err).Error("failed to get JetStream instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.WithField("JetStreamSpec", jetStream.Spec).Info("fetched JetStream instance")

	// Ensuring the NATS JetStream instance is configured correctly
	conn, err := r.NATSClient.Connect(jetStream.Spec.Domain)
	if err != nil {
		log.WithError(err).Error("failed to connect to NATS server")
		return r.updateStatus(ctx, &jetStream, false, err.Error())
	}
	defer func() {
		log.WithField("ConnectionStatus", fmt.Sprintf("%+v", conn.Status())).Info("closing NATS connection")
		conn.Close()
	}()

	log.WithField("ConnectionStatus", fmt.Sprintf("%+v", conn.Status())).Info("successfully connected to NATS server")

	// Get JetStream context
	js, err := r.NATSClient.JetStream(conn)
	if err != nil {
		log.WithError(err).Error("failed to get JetStream context")
		return r.updateStatus(ctx, &jetStream, false, err.Error())
	}

	log.WithField("JetStreamContext", fmt.Sprintf("%+v", js)).Info("successfully obtained JetStream context")

	// Get StreamInfo
	streamInfo, err := js.StreamInfo(jetStream.Spec.Account)
	if err != nil {
		if err == nats.ErrStreamNotFound {
			log.WithError(err).Error("stream not found")
		} else {
			log.WithError(err).Error("failed to get StreamInfo")
			return r.updateStatus(ctx, &jetStream, false, err.Error())
		}
	}

	if streamInfo != nil {
		log.WithField("StreamInfo", streamInfo).Info("fetched StreamInfo")
	} else {
		log.Info("streamInfo is nil, stream needs to be created")
	}

	// Convert retention policy
	retentionPolicy, err := toRetentionPolicy(jetStream.Spec.Config.Retention)
	if err != nil {
		log.WithError(err).Error("invalid retention policy")
		return r.updateStatus(ctx, &jetStream, false, err.Error())
	}

	log.WithField("RetentionPolicy", retentionPolicy).Info("converted retention policy")

	// Configure JetStream
	desiredConfig := &nats.StreamConfig{
		Name:      jetStream.Spec.Account,
		Subjects:  jetStream.Spec.Config.Subjects,
		Replicas:  jetStream.Spec.Config.Replicas,
		Retention: retentionPolicy,
	}

	if streamInfo == nil || !compareStreamConfig(streamInfo.Config, *desiredConfig) {
		if _, err := js.AddStream(desiredConfig); err != nil {
			log.WithError(err).Error("failed to configure JetStream")
			return r.updateStatus(ctx, &jetStream, false, err.Error())
		}
		log.WithField("JetStreamName", desiredConfig.Name).Info("successfully configured JetStream")
	} else {
		log.Info("JetStream configuration is already up-to-date")
	}

	// Update status
	return r.updateStatus(ctx, &jetStream, true, "")
}

func (r *JetStreamReconciler) updateStatus(ctx context.Context, natsJetStream *appsv1alpha1.JetStream, configApplied bool, errorMessage string) (ctrl.Result, error) {
	natsJetStream.Status.ConfigApplied = configApplied
	natsJetStream.Status.ErrorMessage = errorMessage
	natsJetStream.Status.LastUpdated = metav1.Now()
	if err := r.KubeClient.Status().Update(ctx, natsJetStream); err != nil {
		log.WithError(err).Error("failed to update JetStream status")
		return ctrl.Result{}, err
	}
	log.WithField("ConfigApplied", configApplied).WithField("ErrorMessage", errorMessage).Info("updated JetStream status")
	return ctrl.Result{}, nil
}

func compareStreamConfig(current, desired nats.StreamConfig) bool {
	if current.Name != desired.Name {
		return false
	}
	if len(current.Subjects) != len(desired.Subjects) {
		return false
	}
	for i, subject := range current.Subjects {
		if subject != desired.Subjects[i] {
			return false
		}
	}
	if current.Replicas != desired.Replicas {
		return false
	}
	if current.Retention != desired.Retention {
		return false
	}
	return true
}

func toRetentionPolicy(policy string) (nats.RetentionPolicy, error) {
	switch policy {
	case "limits":
		return nats.LimitsPolicy, nil
	case "interest":
		return nats.InterestPolicy, nil
	case "workqueue":
		return nats.WorkQueuePolicy, nil
	default:
		return nats.LimitsPolicy, errors.New("invalid retention policy")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *JetStreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.JetStream{}).
		Complete(r)
}
