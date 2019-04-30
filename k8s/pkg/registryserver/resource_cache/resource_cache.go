package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
	"reflect"
)

const (
	NseResource = "networkserviceendpoints"
	NsResource  = "networkservices"
	NsmResource = "networkservicemanagers"
)

type cacheConfig struct {
	keyFunc             func(obj interface{}) string
	resourceAddedFunc   func(obj interface{})
	resourceUpdatedFunc func(obj interface{})
	resourceGetFunc     func(key string) interface{}
	resourceDeletedFunc func(key string)
	resourceType        string
}

type abstractResourceCache struct {
	eventCh chan resourceEvent
	config  cacheConfig
}

const defaultChannelSize = 40

func newAbstractResourceCache(config cacheConfig) abstractResourceCache {
	return abstractResourceCache{
		eventCh: make(chan resourceEvent, defaultChannelSize),
		config:  config,
	}
}

func (c *abstractResourceCache) start(informerFactory externalversions.SharedInformerFactory) (func(), error) {
	informerStopCh, err := c.startInformer(informerFactory)
	if err != nil {
		return nil, err
	}
	cacheStopCh := make(chan struct{})
	go c.run(cacheStopCh)
	stopFunc := func() {
		close(informerStopCh)
		close(cacheStopCh)
	}
	return stopFunc, nil
}

func (c *abstractResourceCache) add(obj interface{}) {
	c.eventCh <- resourceAddEvent{obj}
}

func (c *abstractResourceCache) get(key string) interface{} {
	getCh := make(chan interface{})
	c.eventCh <- resourceGetEvent{key, getCh}
	return <-getCh
}
func (c *abstractResourceCache) update(obj interface{}) {
	c.eventCh <- resourceUpdateEvent{obj}
}

func (c *abstractResourceCache) delete(key string) {
	c.eventCh <- resourceDeleteEvent{key}
}

func (c *abstractResourceCache) syncExec(f func()) {
	doneCh := make(chan struct{})
	c.eventCh <- syncExecEvent{f, doneCh}
	<-doneCh
}

func (c *abstractResourceCache) run(stopCh chan struct{}) {
	for {
		select {
		case e := <-c.eventCh:
			e.accept(c.config)
		case <-stopCh:
			return
		}
	}
}

func (c *abstractResourceCache) startInformer(informerFactory externalversions.SharedInformerFactory) (chan struct{}, error) {
	genericInformer, err := informerFactory.ForResource(v1.SchemeGroupVersion.WithResource(c.config.resourceType))
	if err != nil {
		return nil, err
	}

	informer := genericInformer.Informer()
	c.addEventHandlers(informer)

	stopCh := make(chan struct{})
	go informer.Run(stopCh)
	return stopCh, nil
}

func (c *abstractResourceCache) addEventHandlers(informer cache.SharedInformer) {
	var addFunc func(obj interface{})
	if c.config.resourceAddedFunc != nil {
		addFunc = func(obj interface{}) {
			logrus.Infof("Add from k8s-registry: %v", reflect.TypeOf(obj))
			c.add(obj)
		}
	}

	var updateFunc func(old interface{}, new interface{})
	if c.config.resourceUpdatedFunc != nil {
		updateFunc = func(old interface{}, new interface{}) {
			logrus.Infof("Update from k8s-registry: %v", reflect.TypeOf(old))
			if _, ok := old.(*v1.NetworkServiceManager); !ok {
				return
			}
			logrus.Infof("Old NSM: %v", old.(*v1.NetworkServiceManager))
			logrus.Infof("New NSM: %v", new.(*v1.NetworkServiceManager))
			c.update(new)
		}
	}

	var deleteFunc func(obj interface{})
	if c.config.resourceDeletedFunc != nil {
		deleteFunc = func(obj interface{}) {
			logrus.Infof("Delete from k8s-registry: %v", reflect.TypeOf(obj))
			c.delete(c.config.keyFunc(obj))
		}
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc,
		UpdateFunc: updateFunc,
		DeleteFunc: deleteFunc,
	})
}
