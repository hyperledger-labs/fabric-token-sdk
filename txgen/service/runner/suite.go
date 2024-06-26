/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package runner

import (
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/api"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
	"github.com/sourcegraph/conc"
)

func NewSuiteRunner(testCaseRunner *TestCaseRunner, intermediary *user.IntermediaryClient, metricsReporter metrics.Reporter, logger logging.ILogger) *SuiteRunner {
	return &SuiteRunner{
		logger:          logger,
		intermediary:    intermediary,
		testCaseRunner:  testCaseRunner,
		metricsReporter: metricsReporter,
		customers:       make(map[string]*customerState),
		done:            make(chan struct{}),
	}
}

type SuiteRunner struct {
	logger          logging.ILogger
	customers       map[string]*customerState
	intermediary    *user.IntermediaryClient
	testCaseRunner  *TestCaseRunner
	metricsReporter metrics.Reporter
	done            chan struct{}
}

// TODO collect all transactions and verify that they are equal to stored in the system

func (r *SuiteRunner) Run(suiteConfigs []model.SuiteConfig) api.Error {
	r.logger.Infof("Init customer states")
	if err := r.initCustomerStates(suiteConfigs); err != nil {
		return err
	}
	r.logger.Infof("Found %d customers: %v", len(r.customers), r.customers)

	r.logger.Infof("Launch throughput logger")
	go r.printTPS()

	r.logger.Infof("Start suite execution")
	for _, suite := range suiteConfigs {
		r.runSuite(suite)
	}

	r.logger.Infof("Check customer balances")
	r.checkCustomerBalances()
	r.logger.Infof(r.metricsReporter.Summary())

	close(r.done)

	time.Sleep(time.Second)

	return nil
}

func (r *SuiteRunner) initCustomerStates(suiteConfigs []model.SuiteConfig) api.Error {
	for _, u := range collectUsers(suiteConfigs) {
		balance, err := r.intermediary.GetBalance(u)
		if err != nil {
			return err
		}
		r.customers[u] = &customerState{Name: u, StartingAmount: balance}
	}
	return nil
}

func (r *SuiteRunner) runSuite(suite model.SuiteConfig) {
	wg := conc.NewWaitGroup()
	r.logger.Infof("========================== Start suite %s ==========================", suite.Name)
	for i := 0; i < suite.Iterations; i++ {
		for _, testCase := range suite.Cases {
			wg.Go(func() {
				result := r.testCaseRunner.Run(&testCase, r.customers, &TestCaseSettings{
					Iteration:        i + 1,
					CallsDelay:       suite.Delay,
					ExecuteIssuance:  true,
					PoolSize:         suite.PoolSize,
					UseExistingFunds: suite.UseExistingFunds,
				})
				if !result.Success {
					r.logger.Errorf("Test case failed: %s", result.Error.Error())
				}
			})
		}
		wg.Wait()
	}
	r.logger.Infof("========================== End suite %s ==========================", suite.Name)
}

func collectUsers(suiteConfigs []model.SuiteConfig) []model.UserAlias {
	usernameMap := make(map[string]struct{})
	for _, s := range suiteConfigs {
		for _, c := range s.Cases {
			for _, username := range append(c.Payees, c.Payer) {
				usernameMap[username] = struct{}{}
			}
		}
	}

	usernames := make([]model.UserAlias, 0, len(usernameMap))
	for username := range usernameMap {
		usernames = append(usernames, username)
	}

	return usernames
}

func (r *SuiteRunner) printTPS() {
	activeRequestReportingInterval := time.NewTicker(time.Millisecond * 500)

	totalReqReportingInterval := time.NewTicker(3 * time.Second)

	for {
		select {
		case <-totalReqReportingInterval.C:
			r.logger.Infof(r.metricsReporter.GetTotalRequests())
		case <-activeRequestReportingInterval.C:
			r.logger.Infof(r.metricsReporter.GetActiveRequests())
		case <-r.done:
			activeRequestReportingInterval.Stop()
			totalReqReportingInterval.Stop()
			r.logger.Infof("Quitting TPS monitoring... %s", r.metricsReporter.GetActiveRequests())
			return
		}
	}
}

func (r *SuiteRunner) checkCustomerBalances() {
	totalWithdrawn := api.Amount(0)
	totalPaid := api.Amount(0)
	totalReceived := api.Amount(0)
	for _, c := range r.customers {
		r.logger.Infof(
			"Customer: '%s', starting amount: %d, withdrawn amount %d, paid amount %d, received amount %d",
			c.Name, c.StartingAmount, c.WithdrawnAmount, c.PaidAmount, c.ReceivedAmount,
		)

		totalWithdrawn += c.WithdrawnAmount
		totalPaid += c.PaidAmount
		totalReceived += c.ReceivedAmount
	}

	msg := fmt.Sprintf("Total withdrawn: %d, total paid %d", totalWithdrawn, totalPaid)
	if totalWithdrawn == totalPaid {
		r.logger.Infof(msg)
	} else {
		r.logger.Errorf(msg)
	}

	msg = fmt.Sprintf("Total paid: %d, total received %d", totalPaid, totalReceived)
	if totalPaid == totalReceived {
		r.logger.Infof(msg)
	} else {
		r.logger.Errorf(msg)
	}

	for _, c := range r.customers {
		balanceInSystem, err := r.intermediary.GetBalance(c.Name)
		if err != nil {
			r.logger.Errorf("Can't do balance post-analysis of user '%s'", c.Name)
			continue
		}
		r.logger.Infof("Balance of user %s: [%d]", c.Name, balanceInSystem)

		expectedBalance := c.StartingAmount + c.ReceivedAmount + c.WithdrawnAmount - c.PaidAmount

		if balanceInSystem == expectedBalance {
			r.logger.Infof("User's '%s' expected balance %d matches balance in system %d", c.Name, expectedBalance, balanceInSystem)
		} else {
			r.logger.Errorf("User's '%s' expected balance %d doesn't match balance in system %d", c.Name, expectedBalance, balanceInSystem)
		}
	}
}
