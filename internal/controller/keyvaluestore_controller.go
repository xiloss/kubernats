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
	"fmt"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1alpha1 "github.com/xiloss/kubernats/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KeyValueStoreReconciler reconciles a KeyValueStore object
type KeyValueStoreReconciler struct {
	KubeClient client.Client
	Scheme     *runtime.Scheme
	NATSClient *ClientImpl
}

// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=keyvaluestores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=keyvaluestores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=keyvaluestores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeyValueStore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *KeyValueStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)

	logger := log.WithFields(log.Fields{
		"namespace": req.Namespace,
		"name":      req.Name,
	})

	log.Info("Starting reconciliation")

	// Fetch the NATSKeyValueStore instance
	var kvStore appsv1alpha1.KeyValueStore
	if err := r.KubeClient.Get(ctx, req.NamespacedName, &kvStore); err != nil {
		logger.WithError(err).Error("failed to get KeyValueStore instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.WithField("KeyValueStoreSpec", kvStore.Spec).Info("fetched KeyValueStore instance")

	// Connect to NATS server
	conn, err := r.NATSClient.Connect(kvStore.Spec.Endpoint)
	if err != nil {
		log.WithError(err).Error("failed to connect to NATS server")
		return r.updateStatus(ctx, &kvStore, false, err.Error())
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
		return r.updateStatus(ctx, &kvStore, false, err.Error())
	}

	log.WithField("JetStreamContext", fmt.Sprintf("%+v", js)).Info("successfully obtained JetStream context")

	// Ensure the key-value buckets are created or updated
	for _, bucket := range kvStore.Spec.Buckets {
		err := r.ensureBucket(js, &bucket)
		if err != nil {
			log.WithError(err).Error("failed to create or update bucket")
			return r.updateStatus(ctx, &kvStore, false, err.Error())
		}
		log.WithField("BucketName", bucket.Name).Info("successfully ensured bucket")
	}

	// Update status
	return ctrl.Result{}, nil
}

// ensureBucket ensures the key-value bucket is created or updated
func (r *KeyValueStoreReconciler) ensureBucket(js nats.JetStreamContext, bucket *appsv1alpha1.BucketConfig) error {
	_, err := js.KeyValue(bucket.Name)
	if err == nats.ErrBucketNotFound {
		_, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket:       bucket.Name,
			History:      uint8(bucket.History),
			MaxValueSize: int32(bucket.MaxValueSize),
			TTL:          bucket.TTL.Duration,
			Replicas:     bucket.Replicas,
		})
	} else if err == nil {
		log.WithField("BucketName", bucket.Name).Info("bucket already exists")
	}
	return err
}

func (r *KeyValueStoreReconciler) updateStatus(ctx context.Context,
	kvStore *appsv1alpha1.KeyValueStore, configApplied bool, errorMessage string) (ctrl.Result, error) {
	kvStore.Status.Applied = configApplied
	kvStore.Status.ErrorMessage = errorMessage
	kvStore.Status.LastUpdated = metav1.Now()
	if err := r.KubeClient.Status().Update(ctx, kvStore); err != nil {
		log.WithError(err).Error("failed to update KeyValueStore status")
		return ctrl.Result{}, err
	}
	log.WithField("ConfigApplied", configApplied).
		WithField("ErrorMessage", errorMessage).
		Info("updated KeyValueStore status")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeyValueStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.KeyValueStore{}).
		Complete(r)
}
