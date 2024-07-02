/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package model

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/api"
)

type Configuration struct {
	App          AppConfig          `yaml:"app"`
	Suites       []SuiteConfig      `yaml:"suites"`
	UserProvider UserProviderConfig `yaml:"userProvider"`
	Intermediary IntermediaryConfig `yaml:"intermediary"`
}

type AppConfig struct {
	Logging   LogLevel  `yaml:"logging"`
	LogFormat LogFormat `yaml:"logFormat"`
}

type UserProviderConfig struct {
	Users      []UserConfig     `yaml:"users"`
	HttpClient HttpClientConfig `yaml:"httpClient"`
}

type HttpClientConfig struct {
	Timeout             time.Duration `yaml:"timeout"`
	MaxConnsPerHost     int           `yaml:"maxConnsPerHost"`
	MaxIdleConnsPerHost int           `yaml:"maxIdleConnsPerHost"`
}

type Username = string
type UserAlias = string

type UserConfig struct {
	Name     UserAlias `yaml:"name"`
	Username Username  `yaml:"username"`
	Password string    `yaml:"password"`
	Endpoint string    `yaml:"endpoint"`
}

type SuiteConfig struct {
	Name             string        `yaml:"name"`
	PoolSize         int           `yaml:"poolSize"`
	Iterations       int           `yaml:"iterations"`
	Delay            time.Duration `yaml:"delay"`
	Cases            []TestCase    `yaml:"cases"`
	UseExistingFunds bool          `yaml:"useExistingFunds"`
}

type IntermediaryConfig struct {
	DelayAfterInitiation time.Duration `yaml:"delayAfterInitiation"`
}

type IssueConfig struct {
	Total        api.Amount   `yaml:"total"`
	Distribution Distribution `yaml:"distribution"`
	Execute      bool         `yaml:"execute"`
}

type TransferConfig struct {
	Distribution Distribution `yaml:"distribution"`
	Execute      bool         `yaml:"execute"`
}

type TestCase struct {
	Name     string         `yaml:"name"`
	Payer    UserAlias      `yaml:"payer"`
	Payees   []UserAlias    `yaml:"payees"`
	Issue    IssueConfig    `yaml:"issue"`
	Transfer TransferConfig `yaml:"transfer"`
}

// LogLevel String defining a log level.
type LogLevel string

// LogFormat String defining a log format.
type LogFormat string
