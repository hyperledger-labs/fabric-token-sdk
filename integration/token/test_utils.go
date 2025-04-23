/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	postgres2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/common"
	"github.com/onsi/gomega"
)

type ReplicationOpts interface {
	For(name string) []node.Option
}

const None = 0

type ReplicationOptions struct {
	*integration.ReplicationOptions
}

func (o *ReplicationOptions) For(name string) []node.Option {
	opts := o.ReplicationOptions.For(name)
	if _, ok := o.SQLConfigs[name]; ok {
		opts = append(opts, fabric.WithPostgresPersistenceNames(common2.DefaultPersistence, common.AllPrefixes...))
	} else {
		opts = append(opts, fabric.WithSqlitePersistences(common.AllPrefixes...))
	}
	return opts
}

func NoReplication() (*ReplicationOptions, *ReplicaSelector) {
	return &ReplicationOptions{ReplicationOptions: integration.NoReplication}, &ReplicaSelector{}
}

func NewReplicationOptions(factor int, names ...string) (*ReplicationOptions, *ReplicaSelector) {
	if factor < None {
		panic("wrong factor")
	}
	if factor == None {
		return NoReplication()
	}
	replicationFactors := make(ReplicaSelector, len(names))
	sqlConfigs := make(map[string]*postgres2.ContainerConfig, len(names))
	for _, name := range names {
		replicationFactors[name] = factor
		sqlConfigs[name] = postgres2.DefaultConfig(fmt.Sprintf("%s-db", name))

	}
	return &ReplicationOptions{ReplicationOptions: &integration.ReplicationOptions{
		ReplicationFactors: replicationFactors,
		SQLConfigs:         sqlConfigs,
	}}, &replicationFactors
}

type NodeReference struct {
	name          string
	replicaIdx    int
	totalReplicas int
}

func (r *NodeReference) String() string {
	return fmt.Sprintf("node [%s:%d:%d]", r.name, r.replicaIdx, r.totalReplicas)
}

func (r *NodeReference) ReplicaName() string {
	if r.totalReplicas <= 1 {
		return r.name
	}
	r.replicaIdx = (r.replicaIdx + 1) % r.totalReplicas
	return replicaName(r.name, r.replicaIdx)
}

func (r *NodeReference) AllNames() []string {
	if r.totalReplicas <= 1 {
		return []string{r.name}
	}
	replicaNames := make([]string, 0)
	for idx := 0; idx < r.totalReplicas; idx++ {
		replicaNames = append(replicaNames, replicaName(r.name, idx))
	}
	return replicaNames
}

func (r *NodeReference) Id() string { return r.name }

type ReplicaSelector map[string]int

func (s *ReplicaSelector) Get(name string) *NodeReference {
	return &NodeReference{name: name, totalReplicas: (*s)[name]}
}

func (s *ReplicaSelector) All(names ...string) []string {
	replicaNames := make([]string, 0)
	for _, name := range names {
		replicaNames = append(replicaNames, s.Get(name).AllNames()...)
	}
	return replicaNames
}

func AllNames(refs ...*NodeReference) []string {
	replicaNames := make([]string, 0)
	for _, ref := range refs {
		replicaNames = append(replicaNames, ref.AllNames()...)
	}
	return replicaNames
}

func replicaName(name string, idx int) string {
	return fmt.Sprintf("fsc.%s.%d", name, idx)
}

func NewTestSuite(startPort func() int, topologies []api.Topology) *TestSuite {
	return &TestSuite{
		generator: func() (*integration.Infrastructure, error) {
			return integration.New(startPort(), "", topologies...)
		},
	}
}

// NewLocalTestSuite returns a new test suite that stores configuration data in `./testdata` and does not remove it when
// the test is done.
func NewLocalTestSuite(startPort func() int, topologies []api.Topology) *TestSuite {
	return &TestSuite{
		generator: func() (*integration.Infrastructure, error) {
			i, err := integration.New(startPort(), "./testdata", topologies...)
			i.DeleteOnStop = false
			return i, err
		},
	}
}

type TestSuite struct {
	generator func() (*integration.Infrastructure, error)
	II        *integration.Infrastructure
}

func (s *TestSuite) TearDown() {
	s.II.Stop()
}

func (s *TestSuite) Setup() {

	// Create the integration ii
	network, err := s.generator()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	s.II = network
	network.RegisterPlatformFactory(token.NewPlatformFactory())
	network.Generate()
	network.Start()
}
