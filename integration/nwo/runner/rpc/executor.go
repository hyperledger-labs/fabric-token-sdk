/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"context"
	"errors"
	"time"

	runner2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
)

var (
	dummyUserProviderConfig = model.UserProviderConfig{Users: []model.UserConfig{}}
	intermediaryConfig      = model.IntermediaryConfig{DelayAfterInitiation: time.Second}
)

type SuiteExecutor struct {
	*txgen.SuiteExecutor
}

func NewSuiteExecutor(config UserProviderConfig) (*SuiteExecutor, error) {
	var err error
	s, err := txgen.NewSuiteExecutor(dummyUserProviderConfig, intermediaryConfig, model.ServerConfig{Endpoint: config.ControllerEndpoint})

	if err != nil {
		return nil, err
	}

	err = errors.Join(
		s.C.Provide(func() UserProviderConfig { return config }),
		s.C.Provide(newUserProvider),
	)
	if err != nil {
		return nil, err
	}

	err = errors.Join(
		s.C.Decorate(func(_ user.Provider, p *runner2.ViewUserProvider) user.Provider { return p }),
		s.C.Decorate(func(_ runner.SuiteRunner, runner *runner.RestRunner, userProvider *runner2.ViewUserProvider, logger logging.ILogger) runner.SuiteRunner {
			return runner2.NewViewRunner(runner, userProvider, logger, config.Auditors[0].Name, config.Issuers[0].Name)
		}),
	)
	if err != nil {
		return nil, err
	}
	return &SuiteExecutor{SuiteExecutor: s}, nil
}

func (e *SuiteExecutor) Execute(suites []model.SuiteConfig) error {
	return e.C.Invoke(func(s runner.SuiteRunner) error {
		if err := s.Start(context.TODO()); err != nil {
			return err
		}
		s.PushSuites(suites...)

		// Do not shut down
		return nil
	})
}
