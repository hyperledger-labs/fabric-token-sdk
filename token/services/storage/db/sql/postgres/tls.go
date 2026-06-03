/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	fscPostgres "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
)

// TLSConfig defines the configuration parameters for securing database connections.
type TLSConfig struct {
	Enabled      bool   `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	ServerName   string `mapstructure:"server_name" yaml:"server_name" json:"server_name"`
	CertPath     string `mapstructure:"cert_path" yaml:"cert_path" json:"cert_path"`
	KeyPath      string `mapstructure:"key_path" yaml:"key_path" json:"key_path"`
	ClientCACert string `mapstructure:"client_ca_cert" yaml:"client_ca_cert" json:"client_ca_cert"`
	RootCertPath string `mapstructure:"root_cert_path" yaml:"root_cert_path" json:"root_cert_path"`
	SSLMode      string `mapstructure:"ssl_mode" yaml:"ssl_mode" json:"ssl_mode"`
}

// tlsConfigProvider wraps configProvider to handle TLS database option unmarshalling.
type tlsConfigProvider struct {
	wrapped configProvider
	config  driver3.Config
}

// GetOpts unmarshals database options, registers custom TLS settings with pgx standard library if enabled,
// and returns the config options.
func (p *tlsConfigProvider) GetOpts(name driver2.PersistenceName, params ...string) (*fscPostgres.Config, error) {
	opts, err := p.wrapped.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	var tlsConfig TLSConfig
	tlsKey := fmt.Sprintf("fsc.persistences.%s.opts.tls", name)
	if p.config.IsSet(tlsKey) {
		if err := p.config.UnmarshalKey(tlsKey, &tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal database TLS config: %w", err)
		}
	} else {
		defaultTlsKey := "fsc.persistences.default.opts.tls"
		if p.config.IsSet(defaultTlsKey) {
			if err := p.config.UnmarshalKey(defaultTlsKey, &tlsConfig); err != nil {
				return nil, fmt.Errorf("failed to unmarshal default database TLS config: %w", err)
			}
		}
	}

	if tlsConfig.Enabled {
		registeredConnStr, err := RegisterTLSConnection(opts.DataSource, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to register TLS connection config: %w", err)
		}
		opts.DataSource = registeredConnStr
	}

	return opts, nil
}

// RegisterTLSConnection parses the datasource string, configures standard Go TLS,
// and registers the customized pgx connection with the stdlib driver.
func RegisterTLSConnection(dataSource string, tlsCfg TLSConfig) (string, error) {
	connConfig, err := pgx.ParseConfig(dataSource)
	if err != nil {
		return "", fmt.Errorf("failed to parse database datasource: %w", err)
	}

	tlsConfig := &tls.Config{}

	if tlsCfg.ServerName != "" {
		tlsConfig.ServerName = tlsCfg.ServerName
	} else {
		tlsConfig.ServerName = connConfig.Host
	}

	if tlsCfg.RootCertPath != "" {
		caCert, err := os.ReadFile(tlsCfg.RootCertPath)
		if err != nil {
			return "", fmt.Errorf("failed to read root certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return "", fmt.Errorf("failed to append root certificate from PEM")
		}
		tlsConfig.RootCAs = caCertPool
	}

	if tlsCfg.CertPath != "" && tlsCfg.KeyPath != "" {
		cert, err := tls.LoadX509KeyPair(tlsCfg.CertPath, tlsCfg.KeyPath)
		if err != nil {
			return "", fmt.Errorf("failed to load client key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	sslMode := tlsCfg.SSLMode
	switch sslMode {
	case "disable":
		connConfig.TLSConfig = nil
		return dataSource, nil
	case "allow", "prefer":
		connConfig.TLSConfig = tlsConfig
	case "require":
		tlsConfig.InsecureSkipVerify = true
		connConfig.TLSConfig = tlsConfig
	case "verify-ca":
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return fmt.Errorf("no peer certificates presented")
			}
			opts := x509.VerifyOptions{
				DNSName: "",
				Roots:   tlsConfig.RootCAs,
			}
			if len(cs.PeerCertificates) > 1 {
				opts.Intermediates = x509.NewCertPool()
				for _, cert := range cs.PeerCertificates[1:] {
					opts.Intermediates.AddCert(cert)
				}
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			return err
		}
		connConfig.TLSConfig = tlsConfig
	case "verify-full", "":
		tlsConfig.InsecureSkipVerify = false
		connConfig.TLSConfig = tlsConfig
	default:
		return "", fmt.Errorf("unsupported ssl mode: %s", sslMode)
	}

	connStr := stdlib.RegisterConnConfig(connConfig)
	return connStr, nil
}
