package shared

import (
	"reflect"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	kclientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	kinformers "k8s.io/kubernetes/pkg/client/informers/informers_generated/externalversions"
	kinternalinformers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"

	oclient "github.com/openshift/origin/pkg/client"
)

type InformerFactory interface {
	// Start starts informers that can start AFTER the API server and controllers have started
	Start(stopCh <-chan struct{})
	// StartCore starts core informers that must initialize in order for the API server to start
	StartCore(stopCh <-chan struct{})

	ClusterPolicies() ClusterPolicyInformer
	ClusterPolicyBindings() ClusterPolicyBindingInformer
	Policies() PolicyInformer
	PolicyBindings() PolicyBindingInformer

	DeploymentConfigs() DeploymentConfigInformer
	BuildConfigs() BuildConfigInformer
	Builds() BuildInformer
	ImageStreams() ImageStreamInformer
	SecurityContextConstraints() SecurityContextConstraintsInformer
	ClusterResourceQuotas() ClusterResourceQuotaInformer

	KubernetesInformers() kinformers.SharedInformerFactory
	InternalKubernetesInformers() kinternalinformers.SharedInformerFactory
}

// ListerWatcherOverrides allows a caller to specify special behavior for particular ListerWatchers
// For instance, authentication and authorization types need to go direct to etcd, not through an API server
type ListerWatcherOverrides interface {
	// GetListerWatcher returns back a ListerWatcher for a given resource or nil if
	// no particular ListerWatcher was specified for the type
	GetListerWatcher(resource schema.GroupResource) cache.ListerWatcher
}

type DefaultListerWatcherOverrides map[schema.GroupResource]cache.ListerWatcher

func (o DefaultListerWatcherOverrides) GetListerWatcher(resource schema.GroupResource) cache.ListerWatcher {
	return o[resource]
}

func NewInformerFactory(
	internalKubeInformers kinternalinformers.SharedInformerFactory,
	kubeInformers kinformers.SharedInformerFactory,
	kubeClient kclientset.Interface,
	originClient oclient.Interface,
	customListerWatchers ListerWatcherOverrides,
	defaultResync time.Duration,
) InformerFactory {
	return &sharedInformerFactory{
		internalKubeInformers: internalKubeInformers,
		kubeInformers:         kubeInformers,
		kubeClient:            kubeClient,
		originClient:          originClient,
		customListerWatchers:  customListerWatchers,
		defaultResync:         defaultResync,

		informers:            map[reflect.Type]cache.SharedIndexInformer{},
		coreInformers:        map[reflect.Type]cache.SharedIndexInformer{},
		startedInformers:     map[reflect.Type]bool{},
		startedCoreInformers: map[reflect.Type]bool{},
	}
}

type sharedInformerFactory struct {
	internalKubeInformers kinternalinformers.SharedInformerFactory
	kubeInformers         kinformers.SharedInformerFactory
	kubeClient            kclientset.Interface
	originClient          oclient.Interface
	customListerWatchers  ListerWatcherOverrides
	defaultResync         time.Duration

	informers            map[reflect.Type]cache.SharedIndexInformer
	coreInformers        map[reflect.Type]cache.SharedIndexInformer
	startedInformers     map[reflect.Type]bool
	startedCoreInformers map[reflect.Type]bool
	lock                 sync.Mutex
}

func (f *sharedInformerFactory) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for informerType, informer := range f.informers {
		if !f.startedInformers[informerType] {
			go informer.Run(stopCh)
			f.startedInformers[informerType] = true
		}
	}
}

func (f *sharedInformerFactory) StartCore(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for informerType, informer := range f.coreInformers {
		if !f.startedCoreInformers[informerType] {
			go informer.Run(stopCh)
			f.startedCoreInformers[informerType] = true
		}
	}
}

func (f *sharedInformerFactory) ClusterPolicies() ClusterPolicyInformer {
	return &clusterPolicyInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) ClusterPolicyBindings() ClusterPolicyBindingInformer {
	return &clusterPolicyBindingInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) Policies() PolicyInformer {
	return &policyInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) PolicyBindings() PolicyBindingInformer {
	return &policyBindingInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) DeploymentConfigs() DeploymentConfigInformer {
	return &deploymentConfigInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) BuildConfigs() BuildConfigInformer {
	return &buildConfigInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) Builds() BuildInformer {
	return &buildInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) ImageStreams() ImageStreamInformer {
	return &imageStreamInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) SecurityContextConstraints() SecurityContextConstraintsInformer {
	return &securityContextConstraintsInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) ClusterResourceQuotas() ClusterResourceQuotaInformer {
	return &clusterResourceQuotaInformer{sharedInformerFactory: f}
}

func (f *sharedInformerFactory) KubernetesInformers() kinformers.SharedInformerFactory {
	return f.kubeInformers
}

func (f *sharedInformerFactory) InternalKubernetesInformers() kinternalinformers.SharedInformerFactory {
	return f.internalKubeInformers
}
