package k8s_manager

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// createKubernetesClient creates a Kubernetes client
func CreateKubernetesClient() (*kubernetes.Clientset, error) {
	// Try to load from kubeconfig first
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		fmt.Errorf("Failed to load kubeconfig: %v", err)
	}
	// If kubeconfig fails, try in-cluster configuration
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("error loading Kubernetes configuration: %v", err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Errorf("Failed to create clientset: %v the clientset %v", err, clientset)
	}

	// Create the clientset
	return kubernetes.NewForConfig(config)
}
