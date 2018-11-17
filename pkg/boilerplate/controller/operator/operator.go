package operator

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/service-serving-cert-signer/pkg/boilerplate/controller"
)

// key is the singleton key shared by all events
// the value is irrelevant, but pandas are awesome
const key = "🐼"

type Runner interface {
	Run(stopCh <-chan struct{})
}

type Option func(*operator) *operator

func New(name string, sync Syncer, opts ...Option) Runner {
	o := &operator{}

	for _, opt := range opts {
		o = opt(o)
	}

	o.runner = controller.New(name, &wrapper{Syncer: sync}, o.opts...)

	return o
}

type operator struct {
	opts   []controller.Option
	runner controller.Runner
}

func WithInformer(getter controller.InformerGetter, filter Filter) Option {
	return func(o *operator) *operator {
		o.opts = append(o.opts,
			controller.WithInformer(getter, controller.FilterFuncs{
				ParentFunc: func(obj v1.Object) (namespace, name string) {
					return key, key // return our singleton key for all events
				},
				AddFunc:    filter.Add,
				UpdateFunc: filter.Update,
				DeleteFunc: filter.Delete,
			}),
		)
		return o
	}
}

func (o *operator) Run(stopCh <-chan struct{}) {
	// only start one worker because we only have one key in our queue (see Operator.WithInformer)
	// since this operator works on a singleton, it does not make sense to ever run more than one worker
	o.runner.Run(1, stopCh)
}
