// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crds

import (
	"os"

	nsapiv1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	nscrd "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	. "github.com/onsi/gomega"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type NSCRD struct {
	clientset nscrd.Interface
	namespace string
	resource  string
	config    *rest.Config
}

func (nscrd *NSCRD) Create(obj *nsapiv1.NetworkService) (*nsapiv1.NetworkService, error) {
	var result nsapiv1.NetworkService
	err := nscrd.clientset.NetworkservicemeshV1().RESTClient().Post().
		Namespace(nscrd.namespace).Resource(nscrd.resource).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (nscrd *NSCRD) Update(obj *nsapiv1.NetworkService) (*nsapiv1.NetworkService, error) {
	var result nsapiv1.NetworkService
	err := nscrd.clientset.NetworkservicemeshV1().RESTClient().Put().
		Namespace(nscrd.namespace).Resource(nscrd.resource).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (nscrd *NSCRD) Delete(name string, options *metaV1.DeleteOptions) error {
	return nscrd.clientset.NetworkservicemeshV1().RESTClient().Delete().
		Namespace(nscrd.namespace).Resource(nscrd.resource).
		Name(name).Body(options).Do().
		Error()
}

func (nscrd *NSCRD) Get(name string) (*nsapiv1.NetworkService, error) {
	var result nsapiv1.NetworkService
	err := nscrd.clientset.NetworkservicemeshV1().RESTClient().Get().
		Namespace(nscrd.namespace).Resource(nscrd.resource).
		Name(name).Do().Into(&result)
	return &result, err
}

func NewNSCRD(namespace string) (*NSCRD, error) {

	path := os.Getenv("KUBECONFIG")
	if len(path) == 0 {
		path = os.Getenv("HOME") + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", path)
	Expect(err).To(BeNil())

	nsmNamespace := namespace
	client := NSCRD{
		namespace: nsmNamespace,
		resource:  "networkservices",
		config:    config,
	}
	client.clientset, err = nscrd.NewForConfig(config)

	Expect(err).To(BeNil())

	return &client, nil
}

func SecureIntranetConnectivity() *nsapiv1.NetworkService {
	return &nsapiv1.NetworkService{
		TypeMeta: v12.TypeMeta{
			APIVersion: "networkservicemesh.io/v1",
			Kind:       "NetworkService",
		},
		ObjectMeta: v12.ObjectMeta{
			Name: "secure-intranet-connectivity",
		},
		Spec: nsapiv1.NetworkServiceSpec{
			Payload: "IP",
			Matches: []*nsapiv1.Match{
				&nsapiv1.Match{
					SourceSelector: map[string]string{
						"app": "firewall",
					},
					Routes: []*nsapiv1.Destination{
						&nsapiv1.Destination{
							DestinationSelector: map[string]string{
								"app": "vpn-gateway",
							},
						},
					},
				},
				&nsapiv1.Match{
					Routes: []*nsapiv1.Destination{
						&nsapiv1.Destination{
							DestinationSelector: map[string]string{
								"app": "firewall",
							},
						},
					},
				},
			},
		},
	}
}
