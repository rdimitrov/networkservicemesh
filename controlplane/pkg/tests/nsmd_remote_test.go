package tests

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

// Below only tests

func TestNSMDRequestClientRemoteNSMD(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage, defaultClusterConfiguration)
	srv2 := newNSMDFullServer(Worker, storage, defaultClusterConfiguration)
	defer srv.Stop()
	defer srv2.Stop()

	srv.testModel.AddDataplane(testDataplane1)

	srv2.testModel.AddDataplane(testDataplane2)

	// Register in both
	nseReg := srv2.registerFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					connection.NetNsInodeKey:    "10",
					connection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	cross_connections := srv2.serviceRegistry.testDataplaneConnection.connections
	Expect(len(cross_connections)).To(Equal(1))
	logrus.Print("End of test")
}

func TestNSMDCloseCrossConnection(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage, defaultClusterConfiguration)
	srv2 := newNSMDFullServer(Worker, storage, defaultClusterConfiguration)
	defer srv.Stop()
	defer srv2.Stop()
	srv.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		LocalMechanisms: []*connection.Mechanism{
			&connection.Mechanism{
				Type: connection.MechanismType_KERNEL_INTERFACE,
			},
		},
		RemoteMechanisms: []*connection2.Mechanism{
			&connection2.Mechanism{
				Type: connection2.MechanismType_VXLAN,
				Parameters: map[string]string{
					connection2.VXLANVNI:   "1",
					connection2.VXLANSrcIP: "10.1.1.1",
				},
			},
		},
		MechanismsConfigured: true,
	})

	srv2.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		RemoteMechanisms: []*connection2.Mechanism{
			&connection2.Mechanism{
				Type: connection2.MechanismType_VXLAN,
				Parameters: map[string]string{
					connection2.VXLANVNI:   "3",
					connection2.VXLANSrcIP: "10.1.1.2",
				},
			},
		},
		MechanismsConfigured: true,
	})

	// Register in both
	nseReg := srv2.registerFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					connection.NetNsInodeKey:    "10",
					connection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	cross_connection := srv.testModel.GetClientConnection(nsmResponse.Id)
	Expect(cross_connection).ToNot(BeNil())

	destConnectionId := cross_connection.Xcon.GetRemoteDestination().GetId()

	cross_connection2 := srv2.testModel.GetClientConnection(destConnectionId)
	Expect(cross_connection2).ToNot(BeNil())

	//Cross connection successfully created, check it closing
	_, err = nsmClient.Close(context.Background(), nsmResponse)
	Expect(err).To(BeNil())

	//We need to check that xcons have been removed from model
	cross_connection = srv.testModel.GetClientConnection(nsmResponse.Id)
	Expect(cross_connection).To(BeNil())

	cross_connection2 = srv2.testModel.GetClientConnection(destConnectionId)
	Expect(cross_connection2).To(BeNil())

}

func TestNSMDDelayRemoteMechanisms(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage, defaultClusterConfiguration)
	srv2 := newNSMDFullServer(Worker, storage, defaultClusterConfiguration)
	defer srv.Stop()
	defer srv2.Stop()

	srv.testModel.AddDataplane(testDataplane1)

	testDataplane2_2 := &model.Dataplane{
		RegisteredName: "test_data_plane2",
		SocketLocation: "tcp:some_addr",
	}

	srv2.testModel.AddDataplane(testDataplane2_2)

	// Register in both
	nseReg := srv2.registerFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					connection.NetNsInodeKey:    "10",
					connection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	type Response struct {
		nsmResponse *connection.Connection
		err         error
	}
	resultChan := make(chan *Response, 1)

	go func(ctx context.Context, req *networkservice.NetworkServiceRequest) {
		nsmResponse, err := nsmClient.Request(ctx, req)
		resultChan <- &Response{nsmResponse: nsmResponse, err: err}
	}(context.Background(), request)

	<-time.Tick(1 * time.Second)

	testDataplane2_2.LocalMechanisms = testDataplane2.LocalMechanisms
	testDataplane2_2.RemoteMechanisms = testDataplane2.RemoteMechanisms
	testDataplane2_2.MechanismsConfigured = true

	res := <-resultChan
	Expect(res.err).To(BeNil())
	Expect(res.nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for crМфвук31oss connections.
	cross_connections := srv2.serviceRegistry.testDataplaneConnection.connections
	Expect(len(cross_connections)).To(Equal(1))
	logrus.Print("End of test")
}
