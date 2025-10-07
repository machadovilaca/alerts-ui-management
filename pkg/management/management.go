package management

import (
	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
)

type client struct {
	k8sClient k8s.Client
	mapper    mapper.Client
}
