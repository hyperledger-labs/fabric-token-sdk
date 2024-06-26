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
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"

	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"

	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/metrics"
	metrics2 "github.com/hyperledger/fabric-lib-go/common/metrics"
	"go.uber.org/dig"
)

type Runner struct {
	*runner.SuiteRunner
}

func NewRunner(nw *integration.Infrastructure) (*Runner, error) {
	c := dig.New()

	err := errors.Join(
		c.Provide(func() *integration.Infrastructure { return nw }),
		c.Provide(func() logging.ILogger { return flogging.MustGetLogger("client") }),
		c.Provide(func() model.IntermediaryConfig { return model.IntermediaryConfig{DelayAfterInitiation: time.Second} }),
		c.Provide(func() metrics2.Provider { return &metrics.Provider{} }),
		c.Provide(newUserProvider),
		c.Provide(runner.NewSuiteRunner),
		c.Provide(user.NewIntermediaryClient),
		c.Provide(runner.NewTestCaseRunner),
		c.Provide(func(p metrics2.Provider) (metrics.Collector, metrics.Reporter) {
			c := metrics.NewCollector(p)
			return c, metrics.NewReporter(c)
		}),
	)
	if err != nil {
		return nil, err
	}
	r := &Runner{}
	if err := c.Invoke(func(sr *runner.SuiteRunner) { r.SuiteRunner = sr }); err != nil {
		return nil, err
	}
	return r, nil
}
