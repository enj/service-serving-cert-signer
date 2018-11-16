package controller

import (
	"bytes"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiserviceclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
	apiserviceinformer "k8s.io/kube-aggregator/pkg/client/informers/externalversions/apiregistration/v1"
	apiservicelister "k8s.io/kube-aggregator/pkg/client/listers/apiregistration/v1"

	"github.com/openshift/service-serving-cert-signer/pkg/boilerplate/controller"
	"github.com/openshift/service-serving-cert-signer/pkg/controller/api"
)

type ServiceServingCertUpdateController struct {
	apiServiceClient apiserviceclient.APIServicesGetter
	apiServiceLister apiservicelister.APIServiceLister

	caBundle []byte

	// standard controller loop
	// api services that need to be checked
	controller.Runner
}

func NewAPIServiceCABundleInjector(apiServiceInformer apiserviceinformer.APIServiceInformer, apiServiceClient apiserviceclient.APIServicesGetter, caBundle []byte) *ServiceServingCertUpdateController {
	sc := &ServiceServingCertUpdateController{
		apiServiceClient: apiServiceClient,
		apiServiceLister: apiServiceInformer.Lister(),
		caBundle:         caBundle,
	}

	sc.Runner = controller.New("APIServiceCABundleInjector", sc).
		WithInformer(apiServiceInformer.Informer(), controller.FilterFuncs{
			AddFunc:    api.HasInjectCABundleAnnotation,
			UpdateFunc: api.HasInjectCABundleAnnotationUpdate,
		})

	return sc
}

func (c *ServiceServingCertUpdateController) Key(namespace, name string) (v1.Object, error) {
	return c.apiServiceLister.Get(name)
}

func (c *ServiceServingCertUpdateController) Sync(obj v1.Object) error {
	apiService := obj.(*apiregistrationv1.APIService)

	// check if we need to do anything
	if !api.HasInjectCABundleAnnotation(apiService) {
		return nil
	}
	if bytes.Equal(apiService.Spec.CABundle, c.caBundle) {
		return nil
	}

	// avoid mutating our cache
	apiServiceCopy := apiService.DeepCopy()
	apiServiceCopy.Spec.CABundle = c.caBundle
	_, err := c.apiServiceClient.APIServices().Update(apiServiceCopy)
	return err
}
