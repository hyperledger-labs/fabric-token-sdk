/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nwo

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	runner2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/runner"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
)

func newUserProvider(nw *integration.Infrastructure, metricsCollector metrics.Collector, logger logging.ILogger, auditor model.Username) (*runner2.ViewUserProvider, error) {
	fscTopology, err := getFscTopology(nw.Topologies)
	if err != nil {
		return nil, err
	}
	selector := make(token2.ReplicaSelector, len(fscTopology.Nodes))
	for _, node := range fscTopology.Nodes {
		selector[node.ID()] = node.Options.ReplicationFactor()
	}
	users := make(map[model.UserAlias][]user.User, len(fscTopology.Nodes))
	for username, replicationFactor := range selector {
		replicas := make([]user.User, replicationFactor)
		for i, replicaName := range selector.Get(username).AllNames() {
			client := nw.Client(replicaName)
			if client == nil {
				return nil, errors2.Errorf("could not find client for %s, only following found: [%v]", replicaName, collections.Keys(nw.Ctx.ViewClients))
			}
			replicas[i] = runner2.NewViewUser(username, auditor, client, nw, metricsCollector, logger)
		}
		users[username] = replicas
	}
	return runner2.NewViewUserProvider(users), nil
}

func getFscTopology(topologies []api2.Topology) (*fsc.Topology, error) {
	for _, t := range topologies {
		if fscTopology, ok := t.(*fsc.Topology); ok {
			return fscTopology, nil
		}
	}
	return nil, errors.New("fsc topology not found")
}
