/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txgen

import (
	"context"
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

// SuiteExecutor instantiates all dependencies and creates a suite runner
type SuiteExecutor struct {
	C *dig.Container // Allow for overwriting dependencies
}

func NewSuiteExecutor(userProviderConfig model.UserProviderConfig, intermediaryConfig model.IntermediaryConfig, serverConfig model.ServerConfig) (*SuiteExecutor, api.Error) {
	c := dig.New()

	err := errors.Join(
		c.Provide(func() logging.ILogger { return logging.MustGetLogger("client") }),
		c.Provide(func() model.IntermediaryConfig { return intermediaryConfig }),
		c.Provide(func() model.UserProviderConfig { return userProviderConfig }),
		c.Provide(metrics.NewProvider),
		c.Provide(rest.NewRestUserProvider),
		c.Provide(runner.NewBase),
		c.Provide(func(r *runner.BaseRunner, config model.ServerConfig, logger logging.ILogger) *runner.RestRunner {
			return runner.NewRest(r, config, logger)
		}),
		c.Provide(func(r *runner.RestRunner) runner.SuiteRunner { return r }),
		c.Provide(func() model.ServerConfig { return serverConfig }),
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
	return &SuiteExecutor{C: c}, nil
}

func (r *SuiteExecutor) Execute(suites []model.SuiteConfig) error {
	return r.C.Invoke(func(s runner.SuiteRunner) error {
		if err := s.Start(context.TODO()); err != nil {
			return err
		}
		s.PushSuites(suites...)
		return s.ShutDown()
	})
}
