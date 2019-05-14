// Copyright 2018, 2019 VMware, Inc.
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

package common

import (
	"strconv"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	advertiseNseNameEnv   = "ADVERTISE_NSE_NAME"
	advertiseNseLabelsEnv = "ADVERTISE_NSE_LABELS"
	outgoingNscNameEnv    = "OUTGOING_NSC_NAME"
	outgoingNscLabelsEnv  = "OUTGOING_NSC_LABELS"
	tracerEnabled         = "TRACER_ENABLED"
	mechanismTypeEnv      = "MECHANISM_TYPE"
	ipAddressEnv          = "IP_ADDRESS"
)

// NSConfiguration contains the full configuration used in the SDK
type NSConfiguration struct {
	NsmServerSocket    string
	NsmClientSocket    string
	Workspace          string
	AdvertiseNseName   string
	OutgoingNscName    string
	AdvertiseNseLabels string
	OutgoingNscLabels  string
	TracerEnabled      bool
	MechanismType      string
	IPAddress          string
}

// CompleteNSConfiguration fills all unset options from the env variables
func (configuration *NSConfiguration) CompleteNSConfiguration() {

	if len(configuration.NsmServerSocket) == 0 {
		configuration.NsmServerSocket = getEnv(nsmd.NsmServerSocketEnv, "nsmServerSocket", true)
	}

	if len(configuration.NsmClientSocket) == 0 {
		configuration.NsmClientSocket = getEnv(nsmd.NsmClientSocketEnv, "nsmClientSocket", true)
	}

	if len(configuration.Workspace) == 0 {
		configuration.Workspace = getEnv(nsmd.WorkspaceEnv, "workspace", true)
	}

	if len(configuration.AdvertiseNseName) == 0 {
		configuration.AdvertiseNseName = getEnv(advertiseNseNameEnv, "Advertise Network Service Name", false)
	}

	if len(configuration.OutgoingNscName) == 0 {
		configuration.OutgoingNscName = getEnv(outgoingNscNameEnv, "Outgoing Network Service Name", false)
	}

	if len(configuration.AdvertiseNseLabels) == 0 {
		configuration.AdvertiseNseLabels = getEnv(advertiseNseLabelsEnv, "Advertise labels", false)
	}

	if len(configuration.OutgoingNscLabels) == 0 {
		configuration.OutgoingNscLabels = getEnv(outgoingNscLabelsEnv, "Outgoing labels", false)
	}

	configuration.TracerEnabled, _ = strconv.ParseBool(getEnv(tracerEnabled, "Tracer enabled", false))

	if len(configuration.MechanismType) == 0 {
		configuration.MechanismType = getEnv(mechanismTypeEnv, "Outgoing mechanism type", false)
	}

	if len(configuration.IPAddress) == 0 {
		configuration.IPAddress = getEnv(ipAddressEnv, "IP Address", false)
	}
}

func NSConfigurationFromUrl(configuration *NSConfiguration, url *tools.NSUrl) *NSConfiguration {
	var conf NSConfiguration
	if configuration != nil {
		conf = *configuration
	}
	conf.OutgoingNscName = url.NsName
	var labels strings.Builder
	separator := false
	for k, v := range url.Params {
		if separator {
			labels.WriteRune(',')
		} else {
			separator = true
		}
		labels.WriteString(k)
		labels.WriteRune('=')
		labels.WriteString(v[0])
	}
	conf.OutgoingNscLabels = labels.String()
	return &conf
}
