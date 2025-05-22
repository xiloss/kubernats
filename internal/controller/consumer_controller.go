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
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1alpha1 "github.com/xiloss/kubernats/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConsumerReconciler reconciles a Consumer object
type ConsumerReconciler struct {
	KubeClient client.Client
	Scheme     *runtime.Scheme
	NATSClient *ClientImpl
}

// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=consumers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=consumers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=consumers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Consumer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *ConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)
	logger := log.WithFields(log.Fields{
		"namespace": req.Namespace,
		"name":      req.Name,
	})

	log.Info("Starting reconciliation")

	// Fetch the Consumer instance
	var natsConsumer appsv1alpha1.Consumer
	if err := r.KubeClient.Get(ctx, req.NamespacedName, &natsConsumer); err != nil {
		logger.WithError(err).Error("failed to get Consumer instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.WithField("ConsumerSpec", natsConsumer.Spec).Info("fetched Consumer instance")

	// Connect to NATS server
	conn, err := r.NATSClient.Connect(natsConsumer.Spec.Domain)
	if err != nil {
		log.WithError(err).Error("failed to connect to NATS server")
		return r.updateStatus(ctx, &natsConsumer, false, err.Error())
	}
	defer func() {
		log.WithField("ConnectionStatus",
			fmt.Sprintf("%+v", conn.Status())).Info("closing NATS connection")
		conn.Close()
	}()

	log.WithField("ConnectionStatus",
		fmt.Sprintf("%+v", conn.Status())).Info("successfully connected to NATS server")

	// Get JetStream context
	js, err := r.NATSClient.JetStream(conn)
	if err != nil {
		log.WithError(err).Error("failed to get JetStream context")
		return r.updateStatus(ctx, &natsConsumer, false, err.Error())
	}

	log.WithField("JetStreamContext", fmt.Sprintf("%+v", js)).Info("successfully obtained JetStream context")

	// Convert ack policy
	ackPolicy, err := toAckPolicy(natsConsumer.Spec.AckPolicy)
	if err != nil {
		log.WithError(err).Error("invalid ack policy")
		return r.updateStatus(ctx, &natsConsumer, false, err.Error())
	}

	// Ensure the Consumer is created
	consumerConfig := &nats.ConsumerConfig{
		Durable:       natsConsumer.Spec.Durable,
		AckPolicy:     ackPolicy,
		FilterSubject: natsConsumer.Spec.Filter,
		MaxDeliver:    natsConsumer.Spec.MaxDeliver,
	}

	_, err = js.AddConsumer(natsConsumer.Spec.Stream, consumerConfig)
	if err != nil {
		log.WithError(err).Error("failed to add Consumer")
		return r.updateStatus(ctx, &natsConsumer, false, err.Error())
	}

	log.WithField("ConsumerName", natsConsumer.Spec.Durable).Info("successfully added Consumer")

	// Update status
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.Consumer{}).
		Complete(r)
}

func (r *ConsumerReconciler) updateStatus(ctx context.Context, natsConsumer *appsv1alpha1.Consumer, configApplied bool, errorMessage string) (ctrl.Result, error) {
	natsConsumer.Status.ConsumerCreated = configApplied
	natsConsumer.Status.ErrorMessage = errorMessage
	natsConsumer.Status.LastUpdated = metav1.Now()
	if err := r.KubeClient.Status().Update(ctx, natsConsumer); err != nil {
		log.WithError(err).Error("failed to update Consumer status")
		return ctrl.Result{}, err
	}
	log.WithField("ConfigApplied", configApplied).WithField("ErrorMessage", errorMessage).Info("updated Consumer status")
	return ctrl.Result{}, nil
}

func toAckPolicy(policy string) (nats.AckPolicy, error) {
	switch policy {
	case "none":
		return nats.AckNonePolicy, nil
	case "all":
		return nats.AckAllPolicy, nil
	case "explicit":
		return nats.AckExplicitPolicy, nil
	default:
		return nats.AckNonePolicy, errors.New("invalid ack policy")
	}
}
