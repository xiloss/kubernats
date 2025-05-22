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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1alpha1 "github.com/xiloss/kubernats/api/v1alpha1"
)

var _ = Describe("KeyValueStore Controller", func() {
	var (
		kvReconciler       *KeyValueStoreReconciler
		ctx                context.Context
		natsKeyValueStore  *appsv1alpha1.KeyValueStore
		typeNamespacedName types.NamespacedName
		natsURL            string
		k8sClientset       *kubernetes.Clientset
	)

	BeforeEach(func() {
		natsURL = "nats://localhost:4222"
		ctx = context.Background()

		// Create k8s clientset
		cfg, err := config.GetConfig()
		Expect(err).ToNot(HaveOccurred())
		k8sClientset, err = kubernetes.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		log.Info("created k8s clientset")

		// Delete and apply NATS deployment and service
		cmd := exec.Command("kubectl", "delete", "deploy", "nats", "--ignore-not-found=true")
		output, _ := cmd.CombinedOutput()
		log.Printf("deleted nats deployment: %s", string(output))

		cmd = exec.Command("kubectl", "delete", "svc", "nats-service", "--ignore-not-found=true")
		output, _ = cmd.CombinedOutput()
		log.Printf("deleted nats-service: %s", string(output))

		cmd = exec.Command("kubectl", "apply", "-f", "../../config/nats/nats-config.yaml")
		output, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to apply nats-config.yaml: %s", string(output)))
		log.Printf("applied nats ConfigMap")

		cmd = exec.Command("kubectl", "apply", "-f", "../../config/nats/nats-deployment.yaml")
		output, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to apply nats-deployment.yaml: %s", string(output)))
		log.Printf("applied nats deployment")

		// Deploy a busybox pod to check DNS resolution
		cmd = exec.Command("kubectl", "run", "--rm", "-i", "--restart=Never", "--image=busybox", "busybox", "--", "sh", "-c", "until nslookup nats-service.default.svc.cluster.local; do echo waiting for DNS; sleep 2; done")
		output, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("DNS resolution failed: %s", string(output)))
		log.Info("DNS resolution confirmed")

		// Wait for NATS pods to be ready
		Eventually(func() error {
			pods, err := k8sClientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
				LabelSelector: "app=nats",
			})
			if err != nil {
				return err
			}
			if len(pods.Items) == 0 {
				return fmt.Errorf("no pods found")
			}
			for _, pod := range pods.Items {
				for _, condition := range pod.Status.Conditions {
					if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
						return nil
					}
				}
			}
			return fmt.Errorf("pods are not ready")
		}, 10*time.Minute, 5*time.Second).Should(Succeed())
		log.Info("pods are ready")

		// Wait for NATS server to be ready with exponential backoff
		Eventually(func() error {
			var err error
			backoff := time.Second
			for i := 0; i < 20; i++ {
				nc, err := nats.Connect(natsURL)
				if err == nil {
					defer nc.Close()
					return nil
				}
				log.Infof("waiting for NATS server to be ready: %v", err)
				time.Sleep(backoff)
				backoff *= 2
			}
			return err
		}, 10*time.Minute, 5*time.Second).Should(Succeed())
		log.Info("NATS server is ready")

		kvReconciler = &KeyValueStoreReconciler{
			KubeClient: k8sClient,
			Scheme:     k8sClient.Scheme(),
			NATSClient: &ClientImpl{},
		}

		natsKeyValueStore = &appsv1alpha1.KeyValueStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-kvstore-resource",
				Namespace: "default",
			},
			Spec: appsv1alpha1.KeyValueStoreSpec{
				Endpoint: natsURL,
				Buckets: []appsv1alpha1.BucketConfig{
					{
						Name:         "bucket1",
						Replicas:     1,
						MaxValueSize: 1024,
						History:      5,
						TTL:          metav1.Duration{Duration: time.Hour},
					},
				},
			},
		}

		typeNamespacedName = types.NamespacedName{
			Name:      "test-kvstore-resource",
			Namespace: "default",
		}

		By("creating the custom resource for kind KeyValueStore")
		Expect(k8sClient.Create(ctx, natsKeyValueStore)).To(Succeed())
	})

	AfterEach(func() {
		By("cleaning up the created custom resource")
		Expect(k8sClient.Delete(ctx, natsKeyValueStore)).To(Succeed())

		// Clean up NATS deployment and service
		cmd := exec.Command("kubectl", "delete", "-f", "../../config/nats/nats-deployment.yaml", "--ignore-not-found=true")
		output, err := cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to delete nats-deployment.yaml: %s", string(output)))

		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nats-service",
				Namespace: "default",
			},
		}
		err = k8sClient.Delete(ctx, svc)
		if err != nil && !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nats-config",
				Namespace: "default",
			},
		}
		err = k8sClient.Delete(ctx, cm)
		if err != nil && !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	})

	It("should successfully reconcile the resource", func() {
		_, err := kvReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
		Expect(err).ToNot(HaveOccurred())

		updatedKVS := &appsv1alpha1.KeyValueStore{}
		err = k8sClient.Get(ctx, typeNamespacedName, updatedKVS)
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedKVS.Status.Applied).To(BeTrue())
	})
})
