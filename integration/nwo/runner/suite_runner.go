/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package runner

import (
	"errors"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	metrics2 "github.com/hyperledger/fabric-lib-go/common/metrics"
	"github.ibm.com/decentralized-trust-research/e2e-transaction-generator/configuration/logging"
	"github.ibm.com/decentralized-trust-research/e2e-transaction-generator/model"
	"github.ibm.com/decentralized-trust-research/e2e-transaction-generator/service"
	"github.ibm.com/decentralized-trust-research/e2e-transaction-generator/service/metrics"
	"go.uber.org/dig"
)

type Runner struct {
	*service.SuiteRunner
}

func NewRunner(nw *integration.Infrastructure) (*Runner, error) {
	c := dig.New()

	err := errors.Join(
		c.Provide(func() *integration.Infrastructure { return nw }),
		c.Provide(func() logging.ILogger { return flogging.MustGetLogger("client") }),
		c.Provide(func() model.IntermediaryConfig { return model.IntermediaryConfig{DelayAfterInitiation: time.Second} }),
		c.Provide(func() metrics2.Provider { return &metrics.Provider{} }),
		c.Provide(newUserProvider),
		c.Provide(service.NewSuiteRunner),
		c.Provide(service.NewIntermediaryClient),
		c.Provide(service.NewTestCaseRunner),
		c.Provide(func(p metrics2.Provider) (metrics.Collector, metrics.Reporter) {
			c := metrics.NewCollector(p)
			return c, metrics.NewReporter(c)
		}),
	)
	if err != nil {
		return nil, err
	}
	r := &Runner{}
	if err := c.Invoke(func(sr *service.SuiteRunner) { r.SuiteRunner = sr }); err != nil {
		return nil, err
	}
	return r, nil
}
