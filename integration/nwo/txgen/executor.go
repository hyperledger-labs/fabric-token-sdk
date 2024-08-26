/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txgen

import (
	"context"
	"errors"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/logging"
	metrics3 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/metrics"
	runner2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/user"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/user/rest"
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
		c.Provide(metrics3.NewProvider),
		c.Provide(rest.NewRestUserProvider),
		c.Provide(runner2.NewBase),
		c.Provide(func(r *runner2.BaseRunner, config model.ServerConfig, logger logging.ILogger) *runner2.RestRunner {
			return runner2.NewRest(r, config, logger)
		}),
		c.Provide(func(r *runner2.RestRunner) runner2.SuiteRunner { return r }),
		c.Provide(func() model.ServerConfig { return serverConfig }),
		c.Provide(user.NewIntermediaryClient),
		c.Provide(runner2.NewTestCaseRunner),
		c.Provide(func(p metrics2.Provider) (*metrics3.Metrics, metrics3.Reporter) {
			c := metrics3.NewMetrics(p)
			return c, metrics3.NewReporter(c)
		}),
	)
	if err != nil {
		return nil, api.NewInternalServerError(err, err.Error())
	}
	return &SuiteExecutor{C: c}, nil
}

func (r *SuiteExecutor) Execute(suites []model.SuiteConfig) error {
	return r.C.Invoke(func(s runner2.SuiteRunner) error {
		if err := s.Start(context.TODO()); err != nil {
			return err
		}
		s.PushSuites(suites...)
		return s.ShutDown()
	})
}
