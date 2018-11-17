package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Runner interface {
	Run(workers int, stopCh <-chan struct{})
}

type InformerGetter interface {
	Informer() cache.SharedIndexInformer
}

type Option func(*controller) *controller

func New(name string, sync Syncer, opts ...Option) Runner {
	c := &controller{
		name: name,
		sync: sync,
	}

	c = WithRateLimiter(workqueue.DefaultControllerRateLimiter())(c)

	for _, opt := range opts {
		c = opt(c)
	}

	return c
}

type controller struct {
	name string
	sync Syncer

	queue      workqueue.RateLimitingInterface
	maxRetries int

	cacheSyncs []cache.InformerSynced
}

func WithMaxRetries(maxRetries int) Option {
	return func(c *controller) *controller {
		c.maxRetries = maxRetries
		return c
	}
}

func WithRateLimiter(limiter workqueue.RateLimiter) Option {
	return func(c *controller) *controller {
		c.queue = workqueue.NewNamedRateLimitingQueue(limiter, c.name)
		return c
	}
}

func WithInformerSynced(getter InformerGetter) Option {
	return func(c *controller) *controller {
		c.cacheSyncs = append(c.cacheSyncs, getter.Informer().GetController().HasSynced)
		return c
	}
}

func WithInformer(getter InformerGetter, filter Filter) Option {
	return func(c *controller) *controller {
		informer := getter.Informer()
		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				object := metaOrDie(obj)
				if filter.Add(object) {
					glog.V(4).Infof("%s: handling add %s/%s", c.name, object.GetNamespace(), object.GetName())
					c.add(filter, object)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldObject := metaOrDie(oldObj)
				newObject := metaOrDie(newObj)
				if filter.Update(oldObject, newObject) {
					glog.V(4).Infof("%s: handling update %s/%s", c.name, newObject.GetNamespace(), newObject.GetName())
					c.add(filter, newObject)
				}
			},
			DeleteFunc: func(obj interface{}) {
				accessor, err := meta.Accessor(obj)
				if err != nil {
					tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
					if !ok {
						utilruntime.HandleError(fmt.Errorf("could not get object from tombstone: %+v", obj))
						return
					}
					accessor, err = meta.Accessor(tombstone.Obj)
					if err != nil {
						utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not an accessor: %+v", obj))
						return
					}
				}
				if filter.Delete(accessor) {
					glog.V(4).Infof("%s: handling delete %s/%s", c.name, accessor.GetNamespace(), accessor.GetName())
					c.add(filter, accessor)
				}
			},
		})
		return WithInformerSynced(getter)(c)
	}
}

func (c *controller) add(filter Filter, object v1.Object) {
	namespace, name := filter.Parent(object)
	qKey := queueKey{namespace: namespace, name: name}
	c.queue.Add(qKey)
}

func (c *controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	glog.Infof("Starting %s", c.name)
	defer glog.Infof("Shutting down %s", c.name)

	if !cache.WaitForCacheSync(stopCh, c.cacheSyncs...) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	qKey := key.(queueKey)
	defer c.queue.Done(qKey)

	err := c.handleSync(qKey)
	c.handleKey(qKey, err)

	return true
}

func (c *controller) handleSync(key queueKey) error {
	obj, err := c.sync.Key(key.namespace, key.name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return c.sync.Sync(obj)
}

func (c *controller) handleKey(key queueKey, err error) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	retryForever := c.maxRetries <= 0
	if retryForever || c.queue.NumRequeues(key) < c.maxRetries {
		utilruntime.HandleError(fmt.Errorf("%v failed with: %v", key, err))
		c.queue.AddRateLimited(key)
		return
	}

	utilruntime.HandleError(fmt.Errorf("dropping key %v out of the queue: %v", key, err))
	c.queue.Forget(key)
}

type queueKey struct {
	namespace string
	name      string
}

func metaOrDie(obj interface{}) v1.Object {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		panic(err) // this should never happen
	}
	return accessor
}
