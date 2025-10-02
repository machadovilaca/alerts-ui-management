package k8s

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NamespaceInterface defines operations for managing namespaces
type NamespaceInterface interface {
	// GetNamespaces returns a list of all namespaces in the cluster
	GetNamespaces(ctx context.Context) ([]string, error)

	// NamespaceExists checks if a namespace exists in the cluster
	NamespaceExists(ctx context.Context, namespace string) (bool, error)
}

// namespaceManager implements NamespaceInterface
type namespaceManager struct {
	clientset *kubernetes.Clientset
}

// newNamespaceManager creates a new namespace manager
func newNamespaceManager(clientset *kubernetes.Clientset) NamespaceInterface {
	return &namespaceManager{
		clientset: clientset,
	}
}

// GetNamespaces returns a list of all namespaces in the cluster
func (nm *namespaceManager) GetNamespaces(ctx context.Context) ([]string, error) {
	namespaces, err := nm.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var names []string
	for _, ns := range namespaces.Items {
		names = append(names, ns.Name)
	}

	return names, nil
}

// NamespaceExists checks if a namespace exists in the cluster
func (nm *namespaceManager) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	_, err := nm.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check namespace %s: %w", namespace, err)
	}
	return true, nil
}
