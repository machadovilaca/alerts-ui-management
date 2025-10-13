package management

import (
	"strings"

	"k8s.io/apimachinery/pkg/types"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
)

type client struct {
	k8sClient k8s.Client
	mapper    mapper.Client
}

func IsPlatformAlertRule(prId types.NamespacedName) bool {
	return strings.HasPrefix(prId.Name, "openshift-")
}
