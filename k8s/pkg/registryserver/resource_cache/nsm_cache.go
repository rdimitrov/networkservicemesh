package resource_cache

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	. "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkServiceManagerCache struct {
	cache                  abstractResourceCache
	networkServiceManagers map[string]*v1.NetworkServiceManager
}

func NewNetworkServiceManagerCache() *NetworkServiceManagerCache {
	rv := &NetworkServiceManagerCache{
		networkServiceManagers: make(map[string]*v1.NetworkServiceManager),
	}
	config := cacheConfig{
		keyFunc:             getNsmKey,
		resourceAddedFunc:   rv.resourceAdded,
		resourceDeletedFunc: rv.resourceDeleted,
		resourceUpdatedFunc: rv.resourceUpdated,
		resourceGetFunc:     rv.resourceGet,
		resourceType:        NsmResource,
	}
	rv.cache = newAbstractResourceCache(config)
	return rv
}

func (c *NetworkServiceManagerCache) Get(key string) *v1.NetworkServiceManager {
	v := c.cache.get(key)
	if v != nil {
		return v.(*v1.NetworkServiceManager)
	}
	return nil
}

func (c *NetworkServiceManagerCache) Add(nsm *v1.NetworkServiceManager) {
	logrus.Infof("NetworkServiceManagerCache.Add(%v)", nsm)
	c.cache.add(nsm)
}

func (c *NetworkServiceManagerCache) Update(nsm *v1.NetworkServiceManager) {
	logrus.Infof("NetworkServiceManagerCache.Update(%v)", nsm)
	c.cache.update(nsm)
}

func (c *NetworkServiceManagerCache) Delete(key string) {
	logrus.Infof("NetworkServiceManagerCache.Delete(%v)", key)
	c.cache.delete(key)
}

func (c *NetworkServiceManagerCache) Start(f SharedInformerFactory, init ...v1.NetworkServiceManager) (func(), error) {
	c.replace(init)
	return c.cache.start(f)
}

func (c *NetworkServiceManagerCache) StartWithResync(f SharedInformerFactory, cs *versioned.Clientset) (func(), error) {
	l, err := cs.NetworkservicemeshV1().NetworkServiceManagers(namespace.GetNamespace()).List(v12.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list NSMs for cache initialization: %v", err)
	}
	return c.Start(f, l.Items...)
}

func (c *NetworkServiceManagerCache) replace(resources []v1.NetworkServiceManager) {
	c.networkServiceManagers = map[string]*v1.NetworkServiceManager{}
	logrus.Infof("Replacing Network service endpoints with: %v", resources)

	for _, r := range resources {
		c.resourceAdded(&r)
	}
}

func (c *NetworkServiceManagerCache) resourceAdded(obj interface{}) {
	nsm := obj.(*v1.NetworkServiceManager)
	logrus.Infof("NetworkServiceManagerCache.Added(%v)", nsm)
	c.networkServiceManagers[getNsmKey(nsm)] = nsm
}

func (c *NetworkServiceManagerCache) resourceUpdated(obj interface{}) {
	nsm := obj.(*v1.NetworkServiceManager)
	logrus.Infof("NetworkServiceManagerCache.resourceUpdated(%v)", nsm)
	c.networkServiceManagers[getNsmKey(nsm)] = nsm
}

func (c *NetworkServiceManagerCache) resourceDeleted(key string) {
	logrus.Infof("NetworkServiceManagerCache.Deleted(%v=%v)", key, c.networkServiceManagers[key])
	delete(c.networkServiceManagers, key)
}

func (c *NetworkServiceManagerCache) resourceGet(key string) interface{} {
	return c.networkServiceManagers[key]
}

func getNsmKey(obj interface{}) string {
	return obj.(*v1.NetworkServiceManager).Name
}
