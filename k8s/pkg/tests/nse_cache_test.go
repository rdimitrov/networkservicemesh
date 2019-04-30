package tests

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resource_cache"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"testing"
	"time"
)

type fakeRegistry struct {
	externalversions.SharedInformerFactory
	externalversions.GenericInformer
	cache.SharedIndexInformer

	eventHandlers []cache.ResourceEventHandler
}

func (f *fakeRegistry) Run(stopCh <-chan struct{}) {

}

func (f *fakeRegistry) ForResource(resource schema.GroupVersionResource) (externalversions.GenericInformer, error) {
	return f, nil
}

func (f *fakeRegistry) Informer() cache.SharedIndexInformer {
	return f
}

func (f *fakeRegistry) AddEventHandler(handler cache.ResourceEventHandler) {
	f.eventHandlers = append(f.eventHandlers, handler)
}

func (f *fakeRegistry) Add(nse *v1.NetworkServiceEndpoint) {
	logrus.Info(len(f.eventHandlers))
	for _, eh := range f.eventHandlers {
		eh.OnAdd(nse)
	}
}

func (f *fakeRegistry) Delete(nse *v1.NetworkServiceEndpoint) {
	for _, eh := range f.eventHandlers {
		eh.OnDelete(nse)
	}
}

func TestK8sRegistryAdd(t *testing.T) {
	RegisterTestingT(t)

	fakeRegistry := fakeRegistry{}
	nseCache := resource_cache.NewNetworkServiceEndpointCache()

	stopFunc, err := nseCache.Start(&fakeRegistry)

	Expect(stopFunc).ToNot(BeNil())
	Expect(err).To(BeNil())

	nse := newTestNse("nse1", "ns1")
	fakeRegistry.Add(nse)

	endpointList := getEndpoints(nseCache, "ns1", 1)
	Expect(len(endpointList)).To(Equal(1))
	Expect(endpointList[0].Name).To(Equal("nse1"))
}

func TestNseCacheConcurrentModification(t *testing.T) {
	RegisterTestingT(t)
	fakeRegistry := fakeRegistry{}
	c := resource_cache.NewNetworkServiceEndpointCache()

	stopFunc, err := c.Start(&fakeRegistry)
	defer stopFunc()
	Expect(stopFunc).ToNot(BeNil())
	Expect(err).To(BeNil())

	c.Add(newTestNse("nse1", "ns1"))
	c.Add(newTestNse("nse2", "ns2"))

	stopRead := RepeatAsync(func() {
		c.Get("nse1")
		c.Get("nse2")
		c.GetByNetworkService("ms1")

	})
	defer stopRead()
	stopWrite := RepeatAsync(func() {
		c.Delete("nsm2")
		c.Add(newTestNse("nse2", "ns2"))
	})
	defer stopWrite()
	time.Sleep(time.Second * 5)
}
func TestNsmdRegistryAdd(t *testing.T) {
	RegisterTestingT(t)

	fakeRegistry := fakeRegistry{}
	nseCache := resource_cache.NewNetworkServiceEndpointCache()

	stopFunc, err := nseCache.Start(&fakeRegistry)

	Expect(stopFunc).ToNot(BeNil())
	Expect(err).To(BeNil())

	nse := newTestNse("nse1", "ns1")
	nseCache.Add(nse)

	endpointList := getEndpoints(nseCache, "ns1", 1)
	Expect(len(endpointList)).To(Equal(1))
	Expect(endpointList[0].Name).To(Equal("nse1"))
}

func TestRegistryDelete(t *testing.T) {
	RegisterTestingT(t)

	fakeRegistry := fakeRegistry{}
	nseCache := resource_cache.NewNetworkServiceEndpointCache()

	stopFunc, err := nseCache.Start(&fakeRegistry)

	Expect(stopFunc).ToNot(BeNil())
	Expect(err).To(BeNil())

	nse1 := newTestNse("nse1", "ns1")
	nse2 := newTestNse("nse2", "ns1")
	nse3 := newTestNse("nse3", "ns2")

	fakeRegistry.Add(nse1)
	fakeRegistry.Add(nse2)
	fakeRegistry.Add(nse3)

	endpointList1 := getEndpoints(nseCache, "ns1", 2)
	Expect(len(endpointList1)).To(Equal(2))
	endpointList2 := getEndpoints(nseCache, "ns2", 1)
	Expect(len(endpointList2)).To(Equal(1))

	fakeRegistry.Delete(nse3)
	endpointList3 := getEndpoints(nseCache, "ns2", 0)
	Expect(len(endpointList3)).To(Equal(0))
}

func getEndpoints(nseCache *resource_cache.NetworkServiceEndpointCache,
	networkServiceName string, expectedLength int) []*v1.NetworkServiceEndpoint {
	var endpointList []*v1.NetworkServiceEndpoint
	for attempt := 0; attempt < 10; <-time.Tick(300 * time.Millisecond) {
		attempt++
		endpointList = nseCache.GetByNetworkService(networkServiceName)
		if len(endpointList) == expectedLength {
			logrus.Infof("Attempt: %v", attempt)
			break
		}
	}
	return endpointList
}

func newTestNse(name string, networkServiceName string) *v1.NetworkServiceEndpoint {
	return &v1.NetworkServiceEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.NetworkServiceEndpointSpec{
			NetworkServiceName: networkServiceName,
			NsmName:            "nsm1",
		},
		Status: v1.NetworkServiceEndpointStatus{
			State: v1.RUNNING,
		},
	}
}
