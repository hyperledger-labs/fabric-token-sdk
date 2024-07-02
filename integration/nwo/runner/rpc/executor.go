/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"context"
	"errors"
	"time"

	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/web"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/operations"
	web2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/web"
	tracing2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	runner2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
	metrics2 "github.com/hyperledger/fabric-lib-go/common/metrics"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/dig"
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

		// Monitoring
		s.C.Provide(func() *operations.Options {
			return &operations.Options{Metrics: operations.MetricsOptions{Provider: config.Monitoring.MetricsProviderType}}
		}),
		s.C.Provide(web.NewOperationsLogger),
		s.C.Provide(func(logger logging.ILogger) *web2.Server {
			return web2.NewServer(web2.Options{ListenAddress: config.Monitoring.MetricsEndpoint, Logger: logger})
		}),
		s.C.Provide(digutils.Identity[*web2.Server](), dig.As(new(operations.Server))),
		s.C.Provide(operations.NewOperationSystem),
		s.C.Provide(func(o *operations.Options, l operations.OperationsLogger) metrics.Provider {
			return operations.NewMetricsProvider(o.Metrics, l, true)
		}),
		s.C.Provide(func(mp metrics.Provider) (trace.TracerProvider, error) {
			tp, err := tracing.NewTracerProviderFromConfig(tracing.Config{
				Provider: config.Monitoring.TracerExporterType,
				Otpl:     tracing.OtplConfig{Address: config.Monitoring.TracerCollectorEndpoint},
				File:     tracing.FileConfig{Path: config.Monitoring.TracerCollectorFile},
			})
			if err != nil {
				return nil, err
			}
			return tracing2.NewTracerProviderWithBackingProvider(tp, mp), nil
		}),
	)
	if err != nil {
		return nil, err
	}

	err = errors.Join(
		s.C.Decorate(func(_ user.Provider, p *runner2.ViewUserProvider) user.Provider { return p }),
		s.C.Decorate(func(_ runner.SuiteRunner, runner *runner.RestRunner, userProvider *runner2.ViewUserProvider, logger logging.ILogger) runner.SuiteRunner {
			return runner2.NewViewRunner(runner, userProvider, logger, config.Auditors[0].Name, config.Issuers[0].Name)
		}),
		s.C.Decorate(func(_ metrics2.Provider, mp metrics.Provider) metrics2.Provider {
			return runner2.NewMetricsProvider(mp)
		}),
	)
	if err != nil {
		return nil, err
	}

	err = errors.Join(
		s.C.Invoke(func(system *operations.System) error { return system.Start() }),
		s.C.Invoke(func(server *web2.Server) error { return server.Start() }),
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
