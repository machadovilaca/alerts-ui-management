package management

import (
	"k8s.io/apimachinery/pkg/types"
)

type NotFoundError struct {
	ID string
}

func (e NotFoundError) Error() string {
	return "alert rule not found"
}

type NotAllowedError struct {
	PrometheusRule types.NamespacedName
}

func (e NotAllowedError) Error() string {
	return "cannot delete alert rule from a platform-managed PrometheusRule"
}
