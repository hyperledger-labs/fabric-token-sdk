/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
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

func NewTestSuite(sqlConfigs map[string]*sql.PostgresConfig, startPort func() int, topologies []api.Topology) *TestSuite {
	return &TestSuite{
		sqlConfigs: sqlConfigs,
		generator: func() (*integration.Infrastructure, error) {
			return integration.New(startPort(), "", topologies...)
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
