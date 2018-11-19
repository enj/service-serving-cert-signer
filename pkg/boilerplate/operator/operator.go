package operator

import (
	"github.com/openshift/service-serving-cert-signer/pkg/boilerplate/controller"
)

type Runner interface {
	Run(stopCh <-chan struct{})
}

func New(name string, sync Syncer, opts ...Option) Runner {
	o := &operator{}

	for _, opt := range opts {
		opt(o)
	}

	o.runner = controller.New(name, &wrapper{Syncer: sync}, o.opts...)

	return o
}

type operator struct {
	opts   []controller.Option
	runner controller.Runner
}

func (o *operator) Run(stopCh <-chan struct{}) {
	// only start one worker because we only have one key in our queue (see Operator.WithInformer)
	// since this operator works on a singleton, it does not make sense to ever run more than one worker
	o.runner.Run(1, stopCh)
}
