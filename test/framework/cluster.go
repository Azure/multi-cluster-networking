/*
Copyright (c) Microsoft Corporation.
Licensed under the MIT license.
*/

// Package framework provides common functionalities for handling a Kubernertes cluster.
package framework

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// PollInterval defines the interval time for a poll operation.
	PollInterval = 1 * time.Second
	// PollTimeout defines the time after which the poll operation times out.
	PollTimeout = 20 * time.Second
)

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	scheme     *runtime.Scheme
	kubeClient client.Client
	name       string
}

// NewCluster creates Cluster and initalizes its kubernetes client.
func NewCluster(name string, scheme *runtime.Scheme) (*Cluster, error) {
	cluster := &Cluster{
		scheme: scheme,
		name:   name,
	}
	if err := cluster.initClusterClient(); err != nil {
		return nil, err
	}
	return cluster, nil
}

// Name returns the cluster name.
func (c *Cluster) Name() string {
	return c.name
}

// Client returns the kubernetes client.
func (c *Cluster) Client() client.Client {
	return c.kubeClient
}

func (c *Cluster) initClusterClient() error {
	clusterConfig, err := c.fetchClientConfig()
	if err != nil {
		return err
	}

	restConfig, err := clusterConfig.ClientConfig()
	if err != nil {
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	}

	kubeClient, err := client.New(restConfig, client.Options{Scheme: c.scheme})
	if err != nil {
		return err
	}
	c.kubeClient = kubeClient
	return nil
}

func (c *Cluster) fetchClientConfig() (clientcmd.ClientConfig, error) {
	kubeConfig, err := kubeConfig()
	if err != nil {
		return nil, err
	}
	cf := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfig},
		&clientcmd.ConfigOverrides{
			CurrentContext: fmt.Sprintf("%s-admin", c.name),
		})
	return cf, nil
}

func kubeConfig() (string, error) {
	kubeconfigEnvKey := "KUBECONFIG"
	kubeConfigPath := os.Getenv(kubeconfigEnvKey)
	if len(kubeConfigPath) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		kubeConfigPath = filepath.Join(homeDir, "/.kube/config")
	}
	if _, err := os.Stat(kubeConfigPath); err != nil {
		return "", fmt.Errorf("failed to find kubeconfig file %s: %v", kubeConfigPath, err)
	}
	return kubeConfigPath, nil
}
