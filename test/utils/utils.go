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

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
)

const (
	prometheusOperatorVersion = "v0.72.0"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"

	certmanagerVersion = "v1.14.4"
	certmanagerURLTmpl = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"

	natsOperatorVersion = "v0.8.3"
	natsOperatorPrereqs = "https://github.com/nats-io/nats-operator/releases/download/%s/00-prereqs.yaml"
	natsOperatorDeploy  = "https://github.com/nats-io/nats-operator/releases/download/%s/10-deployment.yaml"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)
	_, err := Run(cmd)
	return err
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// LoadImageToKindClusterWithName loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}

// InstallNATSOperator installs the NATS Operator along with prerequisites.
func InstallNATSOperator() error {
	prereqsUrl := fmt.Sprintf(natsOperatorPrereqs, natsOperatorVersion)
	deployUrl := fmt.Sprintf(natsOperatorDeploy, natsOperatorVersion)
	fmt.Fprintf(GinkgoWriter, "Installing NATS Operator\n")

	// Install prerequisites
	if err := applyManifest(prereqsUrl); err != nil {
		return fmt.Errorf("failed to apply NATS Operator prerequisites: %v", err)
	}

	// Install NATS Operator deployment
	if err := applyManifest(deployUrl); err != nil {
		return fmt.Errorf("failed to apply NATS Operator deployment: %v", err)
	}

	fmt.Fprintf(GinkgoWriter, "NATS Operator installed successfully\n")
	return nil
}

// UninstallNATSOperator uninstalls the NATS Operator.
func UninstallNATSOperator() {
	prereqsUrl := fmt.Sprintf(natsOperatorPrereqs, natsOperatorVersion)
	deployUrl := fmt.Sprintf(natsOperatorDeploy, natsOperatorVersion)

	fmt.Fprintf(GinkgoWriter, "Uninstalling NATS Operator\n")

	// Uninstall NATS Operator deployment
	if err := deleteManifest(deployUrl); err != nil {
		warnError(err)
	}

	// Uninstall prerequisites
	if err := deleteManifest(prereqsUrl); err != nil {
		warnError(err)
	}

	fmt.Fprintf(GinkgoWriter, "NATS Operator uninstalled successfully\n")
}

// applyManifest applies a Kubernetes manifest file.
func applyManifest(url string) error {
	cmd := exec.Command("kubectl", "apply", "-f", url)
	_, err := Run(cmd)
	return err
}

// deleteManifest deletes a Kubernetes manifest file.
func deleteManifest(url string) error {
	cmd := exec.Command("kubectl", "delete", "-f", url)
	_, err := Run(cmd)
	return err
}

// CreateKindCluster creates a Kind cluster
func CreateKindCluster(name string) error {
	cmd := exec.Command("kind", "create", "cluster", "--name", name)
	_, err := Run(cmd)
	return err
}

// DeleteKindCluster deletes a Kind cluster
func DeleteKindCluster(name string) error {
	cmd := exec.Command("kind", "delete", "cluster", "--name", name)
	_, err := Run(cmd)
	return err
}
