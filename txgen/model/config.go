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
	Server       ServerConfig       `yaml:"server"`
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
	Name     UserAlias `yaml:"name" json:"name"`
	Username Username  `yaml:"username" json:"username"`
	Password string    `yaml:"password" json:"password"`
	Endpoint string    `yaml:"endpoint" json:"endpoint"`
}

type SuiteConfig struct {
	Name             string        `yaml:"name" json:"name"`
	PoolSize         int           `yaml:"poolSize" json:"poolSize"`
	Iterations       int           `yaml:"iterations" json:"iterations"`
	Delay            time.Duration `yaml:"delay" json:"delay"`
	Cases            []TestCase    `yaml:"cases" json:"cases"`
	UseExistingFunds bool          `yaml:"useExistingFunds" yaml:"useExistingFunds"`
}

type IntermediaryConfig struct {
	DelayAfterInitiation time.Duration `yaml:"delayAfterInitiation"`
}

type ServerConfig struct {
	Endpoint string
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
	Name     string         `yaml:"name" json:"name"`
	Payer    UserAlias      `yaml:"payer" json:"payer"`
	Payees   []UserAlias    `yaml:"payees" json:"payees"`
	Issue    IssueConfig    `yaml:"issue" json:"issue"`
	Transfer TransferConfig `yaml:"transfer" json:"transfer"`
}

// LogLevel String defining a log level.
type LogLevel string

// LogFormat String defining a log format.
type LogFormat string
