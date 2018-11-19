package operator

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/service-serving-cert-signer/pkg/boilerplate/controller"
)

type Syncer interface {
	Key() (v1.Object, error)
	Sync(v1.Object) error
}

var _ controller.Syncer = &wrapper{}

type wrapper struct {
	Syncer
}

func (s *wrapper) Key(namespace, name string) (v1.Object, error) {
	return s.Syncer.Key()
}
