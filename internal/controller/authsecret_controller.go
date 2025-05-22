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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1alpha1 "github.com/xiloss/kubernats/api/v1alpha1"
)

// AuthSecretReconciler reconciles a AuthSecret object
type AuthSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=authsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=authsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.kubernats.ai,resources=authsecrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AuthSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *AuthSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = log.FromContext(ctx)

	// TODO(user): your logic here
	logger := log.WithFields(log.Fields{
		"namespace": req.Namespace,
		"name":      req.Name,
	})

	// Fetch the AuthSecret instance
	var authSecret appsv1alpha1.AuthSecret
	if err := r.Get(ctx, req.NamespacedName, &authSecret); err != nil {
		logger.WithError(err).Error("failed to fetch AuthSecret")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensuring the secret exists with the credentials from AuthSecret
	secretName := types.NamespacedName{
		Name:      authSecret.Name,
		Namespace: authSecret.Namespace,
	}

	// Fetch the Secret
	var secret corev1.Secret
	err := r.Get(ctx, secretName, &secret)
	if err != nil && client.IgnoreNotFound(err) != nil {
		logger.WithError(err).Error("failed to fetch secret")
		authSecret.Status.SecretCreated = false
		authSecret.Status.ErrorMessage = err.Error()
		if statusErr := r.Status().Update(ctx, &authSecret); statusErr != nil {
			logger.WithError(statusErr).Error("failed to update AuthSecret status")
		}
		return ctrl.Result{}, err
	}

	// Check if Secret needs to be created or updated
	createSecret := errors.IsNotFound(err)
	secretData := map[string]string{
		"tls":      authSecret.Spec.TLS,
		"token":    authSecret.Spec.Token,
		"username": authSecret.Spec.Credentials.User,
		"password": authSecret.Spec.Credentials.Password,
	}

	if createSecret {
		// Secret does not exist, create it
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      authSecret.Name,
				Namespace: authSecret.Namespace,
			},
			StringData: secretData,
		}
		if err := r.Create(ctx, &secret); err != nil {
			logger.WithError(err).Error("failed to create secret")
			authSecret.Status.SecretCreated = false
			authSecret.Status.ErrorMessage = err.Error()
			if statusErr := r.Status().Update(ctx, &authSecret); statusErr != nil {
				logger.WithError(statusErr).Error("failed to update AuthSecret status")
			}
			return ctrl.Result{}, err
		}
		logger.Info("secret created successfully")
	} else {
		// Secret exists, then should match the desired state
		updateNeeded := false
		for key, value := range secretData {
			if secret.StringData[key] != value {
				secret.StringData[key] = value
				updateNeeded = true
			}
		}

		if updateNeeded {
			if err := r.Update(ctx, &secret); err != nil {
				logger.WithError(err).Error("failed to update secret")
				authSecret.Status.SecretCreated = false
				authSecret.Status.ErrorMessage = err.Error()
				if statusErr := r.Status().Update(ctx, &authSecret); statusErr != nil {
					logger.WithError(statusErr).Error("failed to update AuthSecret status")
				}
				return ctrl.Result{}, err
			}
			logger.Info("secret updated successfully")
		} else {
			logger.Info("secret is already up-to-date")
		}
	}

	// Update status to indicate success
	authSecret.Status.SecretCreated = true
	authSecret.Status.ErrorMessage = ""
	if err := r.Status().Update(ctx, &authSecret); err != nil {
		logger.WithError(err).Error("failed to update AuthSecret status")
		return ctrl.Result{}, err
	}

	logger.Info("AuthSecret successfully reconciled")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.AuthSecret{}).
		Complete(r)
}
