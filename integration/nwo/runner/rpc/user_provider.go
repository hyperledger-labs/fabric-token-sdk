/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"net"
	"os"
	"time"

	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/client/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/client/web"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/grpc"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	runner2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/user"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type UserProviderConfig struct {
	ConnectionType     ConnectionType   `yaml:"connectionType"`
	Users              []UserConfig     `yaml:"users"`
	Auditors           []UserConfig     `yaml:"auditors"`
	Issuers            []UserConfig     `yaml:"issuers"`
	ControllerEndpoint string           `yaml:"controllerEndpoint"`
	Monitoring         MonitoringConfig `yaml:"monitoring"`
}

type MonitoringConfig struct {
	MetricsProviderType     string             `yaml:"metricsProviderType"`
	MetricsEndpoint         string             `yaml:"metricsEndpoint"`
	TracerExporterType      tracing.TracerType `yaml:"tracerExporterType"`
	TracerCollectorEndpoint string             `yaml:"tracerCollectorEndpoint"`
	TracerCollectorFile     string             `yaml:"tracerCollectorFile"`
	TracerSamplingRatio     float64            `yaml:"tracerSamplingRatio"`
}

func (c *UserProviderConfig) IssuerNames() []model.Username {
	names := make([]model.Username, len(c.Issuers))
	for i, issuer := range c.Issuers {
		names[i] = issuer.Name
	}
	return names
}

type UserConfig struct {
	Name     model.UserAlias `yaml:"name"`
	Host     string          `yaml:"host"`
	CorePath string          `yaml:"corePath"`
}

type ConnectionType string

const (
	REST ConnectionType = "REST"
	GRPC ConnectionType = "GRPC"
)

func newUserProvider(c UserProviderConfig, metrics *metrics.Metrics, tracerProvider trace.TracerProvider, logger logging.ILogger) (*runner2.ViewUserProvider, error) {
	users := make(map[model.UserAlias][]user.User, len(c.Users))
	for _, uc := range append(append(c.Users, c.Auditors...), c.Issuers...) {
		u, err := newUser(uc.CorePath, uc.Host, c.ConnectionType, metrics, tracerProvider, logger, c.Auditors[0].Name)
		if err != nil {
			return nil, err
		}
		users[uc.Name] = []user.User{u}
	}
	return runner2.NewViewUserProvider(users), nil
}

func newUser(corePath string, host string, connType ConnectionType, metrics *metrics.Metrics, tracerProvider trace.TracerProvider, logger logging.ILogger, auditor model.Username) (user.User, error) {
	cfg, cli, err := newClient(corePath, host, connType)
	if err != nil {
		return nil, err
	}
	idResolver, err := newResolver(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create identity resolver")
	}
	return runner2.NewViewUser(cfg.GetString("fsc.id"), auditor, cli, idResolver, metrics, tracerProvider, logger), nil
}

func newClient(corePath string, host string, connType ConnectionType) (driver.ConfigService, api2.ViewClient, error) {
	cfg, err := config.NewProvider(corePath)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not find config provider")
	}
	var cli api2.ViewClient
	if connType == REST {
		cli, err = newWebClient(cfg, host)
	} else {
		cli, err = newGrpcClient(cfg, host)
	}
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create view client")
	}
	return cfg, cli, nil
}

func newGrpcClient(configProvider driver.ConfigService, host string) (api2.ViewClient, error) {
	_, port, err := net.SplitHostPort(configProvider.GetString("fsc.grpc.address"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse address")
	}

	cc := &grpc.ConnectionConfig{
		Address:           fmt.Sprintf("%s:%s", host, port),
		TLSEnabled:        true,
		TLSRootCertFile:   configProvider.GetStringSlice("fsc.web.tls.clientRootCAs.files")[0],
		ConnectionTimeout: 10 * time.Second,
	}

	signer, err := view.NewX509SigningIdentity(
		configProvider.GetPath("fsc.identity.cert.file"),
		configProvider.GetPath("fsc.identity.key.file"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signing identity")
	}
	return view.NewClient(&view.Config{ConnectionConfig: cc}, signer, &hasher{}, noop.NewTracerProvider())
}

type hasher struct{}

func (*hasher) GetHash() hash.Hash {
	return sha256.New()
}

func (*hasher) Hash(msg []byte) ([]byte, error) {
	h := sha256.New()
	h.Write(msg)
	return h.Sum(nil), nil
}

func newWebClient(configProvider driver.ConfigService, host string) (api2.ViewClient, error) {
	_, port, err := net.SplitHostPort(configProvider.GetString("fsc.web.address"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse address")
	}
	return web.NewClient(&web.Config{
		Host:        fmt.Sprintf("%s:%s", host, port),
		CACertPath:  configProvider.GetStringSlice("fsc.web.tls.clientRootCAs.files")[0],
		TLSCertPath: configProvider.GetPath("fsc.web.tls.cert.file"),
		TLSKeyPath:  configProvider.GetPath("fsc.web.tls.key.file"),
	})
}

type resolver struct {
	ids map[model.Username]view2.Identity
}

func newResolver(conf driver.ConfigService) (*resolver, error) {
	var resolvers []*fsc.Resolver
	if err := conf.UnmarshalKey("fsc.endpoint.resolvers", &resolvers); err != nil {
		return nil, err
	}
	ids := make(map[model.Username]view2.Identity)
	for _, r := range resolvers {
		b, err := os.ReadFile(r.Identity.Path)
		if err != nil {
			return nil, err
		}
		ids[r.Name] = b
	}
	return &resolver{ids: ids}, nil
}

func (r *resolver) Identity(username model.Username) view2.Identity {
	return r.ids[username]
}
