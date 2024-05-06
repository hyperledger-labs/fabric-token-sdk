/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	. "github.com/onsi/gomega"
)

type ReplicationOpts interface {
	For(name string) []node.Option
}

type ReplicationOptions struct {
	*integration.ReplicationOptions
}

func (o *ReplicationOptions) For(name string) []node.Option {
	opts := o.ReplicationOptions.For(name)
	if sqlConfig, ok := o.SQLConfigs[name]; ok {
		opts = append(opts, token.WithPostgresPersistence(sqlConfig))
	}
	return opts
}

func NoReplication() (*ReplicationOptions, *ReplicaSelector) {
	return &ReplicationOptions{ReplicationOptions: integration.NoReplication}, &ReplicaSelector{}
}

func NewReplicationOptions(factor int, names ...string) (*ReplicationOptions, *ReplicaSelector) {
	if factor < 1 {
		panic("wrong factor")
	}
	replicationFactors := make(map[string]int, len(names))
	sqlConfigs := make(map[string]*sql.PostgresConfig, len(names))
	replicaSelector := make(ReplicaSelector)
	for _, name := range names {
		replicationFactors[name] = factor
		sqlConfigs[name] = sql.DefaultConfig(fmt.Sprintf("%s-db", name))
		replicaSelector[name] = [2]int{0, factor}
	}
	return &ReplicationOptions{ReplicationOptions: &integration.ReplicationOptions{
		ReplicationFactors: replicationFactors,
		SQLConfigs:         sqlConfigs,
	}}, &replicaSelector
}

const (
	currentIdx    = 0
	totalReplicas = 1
)

type ReplicaSelector map[string][2]int

func (s *ReplicaSelector) Get(name string) string {
	if v, ok := (*s)[name]; ok {
		idx := v[currentIdx]
		v[currentIdx] = (idx + 1) % v[totalReplicas]
		return replicaName(name, idx)
	}
	return name
}

func (s *ReplicaSelector) All(names ...string) []string {
	replicaNames := make([]string, 0)
	for _, name := range names {
		if v, ok := (*s)[name]; ok {
			for idx := 0; idx < v[totalReplicas]; idx++ {
				replicaNames = append(replicaNames, replicaName(name, idx))
			}
		} else {
			replicaNames = append(replicaNames, name)
		}
	}
	return replicaNames
}

func replicaName(name string, idx int) string {
	return fmt.Sprintf("fsc.%s.%d", name, idx)
}

func NewTestSuite(sqlConfigs map[string]*sql.PostgresConfig, startPort func() int, topologies []api.Topology) *TestSuite {
	return &TestSuite{
		sqlConfigs: sqlConfigs,
		generator: func() (*integration.Infrastructure, error) {
			i, err := integration.New(startPort(), "", topologies...)
			return i, err
		},
		closeFunc: func() {},
	}
}

type TestSuite struct {
	sqlConfigs map[string]*sql.PostgresConfig
	generator  func() (*integration.Infrastructure, error)

	closeFunc func()
	II        *integration.Infrastructure
}

func (s *TestSuite) TearDown() {
	s.II.Stop()
	s.closeFunc()
}

func (s *TestSuite) Setup() {
	if len(s.sqlConfigs) > 0 {
		closeFunc, err := sql.StartPostgresWithFmt(s.sqlConfigs)
		Expect(err).NotTo(HaveOccurred())
		s.closeFunc = closeFunc
	}

	// Create the integration ii
	network, err := s.generator()
	Expect(err).NotTo(HaveOccurred())
	s.II = network
	network.RegisterPlatformFactory(token.NewPlatformFactory())
	network.Generate()
	network.Start()
}
