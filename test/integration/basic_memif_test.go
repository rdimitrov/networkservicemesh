// +build basic

package nsmd_integration_tests

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
)

func TestSimpleMemifConnection(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.PrepareDefault()

	nodes := nsmd_test_utils.SetupNodes(k8s, 1, defaultTimeout)

	nsmd_test_utils.DeployVppAgentICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	vppagentNsc := nsmd_test_utils.DeployVppAgentNSC(k8s, nodes[0].Node, "vppagent-nsc", defaultTimeout)
	Expect(true, nsmd_test_utils.IsVppAgentNsePinged(k8s, vppagentNsc))
}
