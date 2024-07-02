/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txgen

import (
	"errors"

	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/api"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user/rest"
	metrics2 "github.com/hyperledger/fabric-lib-go/common/metrics"
	"go.uber.org/dig"
)

type Runner struct {
	*runner.SuiteRunner
	C *dig.Container // Allow for overwriting dependencies
}

func NewRunner(userProviderConfig model.UserProviderConfig, intermediaryConfig model.IntermediaryConfig) (*Runner, api.Error) {
	c := dig.New()

	err := errors.Join(
		c.Provide(func() logging.ILogger { return logging.MustGetLogger("client") }),
		c.Provide(func() model.IntermediaryConfig { return intermediaryConfig }),
		c.Provide(func() model.UserProviderConfig { return userProviderConfig }),
		c.Provide(func() metrics2.Provider { return &metrics.Provider{} }),
		c.Provide(rest.NewRestUserProvider),
		c.Provide(runner.NewSuiteRunner),
		c.Provide(user.NewIntermediaryClient),
		c.Provide(runner.NewTestCaseRunner),
		c.Provide(func(p metrics2.Provider) (metrics.Collector, metrics.Reporter) {
			c := metrics.NewCollector(p)
			return c, metrics.NewReporter(c)
		}),
	)
	if err != nil {
		return nil, api.NewInternalServerError(err, err.Error())
	}
	r := &Runner{C: c}
	if err := c.Invoke(func(sr *runner.SuiteRunner) { r.SuiteRunner = sr }); err != nil {
		return nil, api.NewAuthorizationError(err, err.Error())
	}
	return r, nil
}
