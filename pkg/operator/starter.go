package operator

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	scsclient "github.com/openshift/client-go/servicecertsigner/clientset/versioned"
	scsinformers "github.com/openshift/client-go/servicecertsigner/informers/externalversions"
	"github.com/openshift/client-go/servicecertsigner/informers/externalversions/servicecertsigner/v1alpha1"
	"github.com/openshift/library-go/pkg/operator/status"
)

func RunOperator(clientConfig *rest.Config, stopCh <-chan struct{}) error {
	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return err
	}
	scsClient, err := scsclient.NewForConfig(clientConfig)
	if err != nil {
		return err
	}
	dynamicClient, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		return err
	}

	const defaultResync = 10 * time.Minute

	operatorInformers := scsinformers.NewSharedInformerFactoryWithOptions(scsClient, defaultResync,
		scsinformers.WithTweakListOptions(func(options *v1.ListOptions) {
			options.FieldSelector = fields.OneTermEqualSelector("metadata.name", resourceName).String()
		}),
	)
	kubeInformersNamespaced := informers.NewSharedInformerFactoryWithOptions(kubeClient, defaultResync, informers.WithNamespace(targetNamespaceName))

	clusterOperatorStatus := status.NewClusterOperatorStatusController(
		targetNamespaceName,
		targetNamespaceName,
		dynamicClient,
		&operatorStatusProvider{informer: operatorInformers.Servicecertsigner().V1alpha1().ServiceCertSignerOperatorConfigs()},
	)

	operator := NewServiceCertSignerOperator(
		operatorInformers.Servicecertsigner().V1alpha1().ServiceCertSignerOperatorConfigs(),
		kubeInformersNamespaced,
		scsClient.ServicecertsignerV1alpha1(),
		kubeClient.AppsV1(),
		kubeClient.CoreV1(),
		kubeClient.RbacV1(),
	)

	operatorInformers.Start(stopCh)
	kubeInformersNamespaced.Start(stopCh)

	go operator.Run(1, stopCh)
	go clusterOperatorStatus.Run(1, stopCh)

	<-stopCh
	return fmt.Errorf("stopped")
}

type operatorStatusProvider struct {
	informer v1alpha1.ServiceCertSignerOperatorConfigInformer
}

func (p *operatorStatusProvider) Informer() cache.SharedIndexInformer {
	return p.informer.Informer()
}

func (p *operatorStatusProvider) CurrentStatus() (operatorv1alpha1.OperatorStatus, error) {
	instance, err := p.informer.Lister().Get(resourceName)
	if err != nil {
		return operatorv1alpha1.OperatorStatus{}, err
	}
	return instance.Status.OperatorStatus, nil
}
