/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type serverConfig struct {
	CCID           string
	CCaddress      string
	LogLevel       string
	MetricsEnabled bool
	MetricsServer  string
}

func main() {
	metricsEnabledEnv := os.Getenv("CHAINCODE_METRICS_ENABLED")
	metricsEnabled := false
	if len(metricsEnabledEnv) > 0 {
		var err error
		metricsEnabled, err = strconv.ParseBool(metricsEnabledEnv)
		if err != nil {
			fmt.Printf("Error parsing CHAINCODE_METRICS_ENABLED: %s\n", err)
			os.Exit(1)
		}
	}

	config := serverConfig{
		CCID:           os.Getenv("CHAINCODE_ID"),
		CCaddress:      os.Getenv("CHAINCODE_SERVER_ADDRESS"),
		LogLevel:       os.Getenv("CHAINCODE_LOG_LEVEL"),
		MetricsEnabled: metricsEnabled,
		MetricsServer:  os.Getenv("CHAINCODE_METRICS_SERVER"),
	}
	if len(config.MetricsServer) == 0 {
		config.MetricsServer = "localhost:8125"
	}
	flogging.Init(flogging.Config{
		Format:  "'%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}'",
		LogSpec: "info",
		Writer:  os.Stderr,
	})

	fmt.Printf("metrics server at [%s], enabled [%v]", config.MetricsServer, config.MetricsEnabled)

	if config.CCID == "" || config.CCaddress == "" {
		fmt.Println("CC ID or CC address is empty... Running as usual...")
		if os.Getenv("DEVMODE_ENABLED") != "" {
			fmt.Println("starting up in devmode...")
		}
		err := shim.Start(
			&tcc.TokenChaincode{
				TokenServicesFactory: func(bytes []byte) (tcc.PublicParameters, tcc.Validator, error) {
					ppm, v, err := token.NewServicesFromPublicParams(bytes)
					if err != nil {
						return nil, nil, err
					}
					return ppm.PublicParameters(), v, nil
				},
				MetricsEnabled: config.MetricsEnabled,
				MetricsServer:  config.MetricsServer,
			},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Exiting chaincode: %s", err)
			os.Exit(2)
		}
	} else {
		fmt.Println("Token Chaincode CCID : " + config.CCID)
		fmt.Println("Token Chaincode address : " + config.CCaddress)
		fmt.Println("Running Token Chaincode as service ...")
		server := &shim.ChaincodeServer{
			CCID:    config.CCID,
			Address: config.CCaddress,
			CC: &tcc.TokenChaincode{
				TokenServicesFactory: func(bytes []byte) (tcc.PublicParameters, tcc.Validator, error) {
					ppm, v, err := token.NewServicesFromPublicParams(bytes)
					if err != nil {
						return nil, nil, err
					}
					return ppm.PublicParameters(), v, nil
				},
				LogLevel:       config.LogLevel,
				MetricsEnabled: config.MetricsEnabled,
				MetricsServer:  config.MetricsServer,
			},
			TLSProps: shim.TLSProperties{
				// TODO : enable TLS
				Disabled: true,
			},
		}
		err := server.Start()
		if err != nil {
			fmt.Printf("Error starting Token Chaincode: %s", err)
		}
	}
}
