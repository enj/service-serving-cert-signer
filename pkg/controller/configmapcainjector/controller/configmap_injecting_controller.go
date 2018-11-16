package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	informers "k8s.io/client-go/informers/core/v1"
	kcoreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	listers "k8s.io/client-go/listers/core/v1"

	"github.com/openshift/service-serving-cert-signer/pkg/boilerplate/controller"
	"github.com/openshift/service-serving-cert-signer/pkg/controller/api"
)

// ConfigMapCABundleInjectionController is responsible for injecting a CA bundle into configMaps annotated with
// "service.alpha.openshift.io/inject-cabundle"
type ConfigMapCABundleInjectionController struct {
	configMapClient kcoreclient.ConfigMapsGetter
	configMapLister listers.ConfigMapLister

	ca string

	// standard controller loop
	// configMaps that need to be checked
	controller.Runner
}

func NewConfigMapCABundleInjectionController(configMaps informers.ConfigMapInformer, configMapsClient kcoreclient.ConfigMapsGetter, ca string) *ConfigMapCABundleInjectionController {
	ic := &ConfigMapCABundleInjectionController{
		configMapClient: configMapsClient,
		configMapLister: configMaps.Lister(),
		ca:              ca,
	}

	ic.Runner = controller.New("ConfigMapCABundleInjectionController", ic.key, ic.syncConfigMap).
		WithInformer(configMaps.Informer(), controller.FilterFuncs{
			AddFunc:    api.HasInjectCABundleAnnotation,
			UpdateFunc: api.HasInjectCABundleAnnotationUpdate,
		})

	return ic
}

func (ic *ConfigMapCABundleInjectionController) key(namespace, name string) (v1.Object, error) {
	return ic.configMapLister.ConfigMaps(namespace).Get(name)
}

func (ic *ConfigMapCABundleInjectionController) syncConfigMap(obj v1.Object) error {
	sharedConfigMap := obj.(*corev1.ConfigMap)

	// check if we need to do anything
	if !api.HasInjectCABundleAnnotation(sharedConfigMap) {
		return nil
	}
	// skip updating when the CA bundle is already there
	if data, ok := sharedConfigMap.Data[api.InjectionDataKey]; ok && data == ic.ca {
		return nil
	}

	// make a copy to avoid mutating cache state
	configMapCopy := sharedConfigMap.DeepCopy()

	if configMapCopy.Data == nil {
		configMapCopy.Data = map[string]string{}
	}

	configMapCopy.Data[api.InjectionDataKey] = ic.ca

	_, err := ic.configMapClient.ConfigMaps(configMapCopy.Namespace).Update(configMapCopy)
	return err
}
