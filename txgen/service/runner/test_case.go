/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package runner

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/api"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/user"
	"github.com/sourcegraph/conc/pool"
)

type TestCaseResult struct {
	Name      string
	Iteration int
	Success   bool
	Duration  time.Duration
	Error     error
}

type TestCaseSettings struct {
	Iteration        int
	CallsDelay       time.Duration
	ExecuteIssuance  bool
	PoolSize         int
	UseExistingFunds bool
}

func NewTestCaseRunner(intermediary *user.IntermediaryClient, logger logging.ILogger) *TestCaseRunner {
	return &TestCaseRunner{
		logger:       logger,
		intermediary: intermediary,
	}
}

type TestCaseRunner struct {
	logger       logging.ILogger
	intermediary *user.IntermediaryClient
}

func (r *TestCaseRunner) Run(scenario *model.TestCase, customers map[string]*customerState, settings *TestCaseSettings) *TestCaseResult {
	r.logger.Infof("Starting case %s", scenario.Name)
	payer := customers[scenario.Payer]
	funds := scenario.Issue.Total

	if settings.UseExistingFunds {
		r.logger.Infof("Use existing funds enabled. Check the balance of %s", payer.Name)
		currentBalance, err := r.intermediary.GetBalance(payer.Name)
		if err != nil {
			return &TestCaseResult{
				Success:   false,
				Name:      scenario.Name,
				Iteration: settings.Iteration,
				Error:     err,
			}
		}
		funds = currentBalance
		r.logger.Infof("User [%s] has balance: [%d]", payer.Name, currentBalance)
	}

	withdrawAmnts, err := scenario.Issue.Distribution.GetAmounts(funds)
	if err != nil {
		r.logger.Errorf("Can't generate withdraw amounts: %s", err.GetMessage())
		return &TestCaseResult{
			Success:   false,
			Name:      scenario.Name,
			Iteration: settings.Iteration,
			Error:     err,
		}
	}
	r.logger.Infof("%d withdrawal amounts: %v", len(withdrawAmnts), withdrawAmnts)

	transferAmnts, err := scenario.Transfer.Distribution.GetAmounts(funds)
	if err != nil {
		r.logger.Errorf("Can't generate transfer amounts: %s", err.GetMessage())
		return &TestCaseResult{
			Success:   false,
			Name:      scenario.Name,
			Iteration: settings.Iteration,
			Error:     err,
		}
	}
	r.logger.Infof("%d transfer amounts: %v", len(transferAmnts), transferAmnts)

	start := time.Now()
	r.logger.Infof("============= Start test case %s, iter %d =============", scenario.Name, settings.Iteration)

	if scenario.Issue.Execute && !settings.UseExistingFunds {
		r.logger.Infof("Starting withdrawals")
		execErr := r.doWithdrawals(payer, withdrawAmnts, settings)

		if execErr != nil {
			return &TestCaseResult{
				Name:      scenario.Name,
				Success:   false,
				Duration:  time.Since(start),
				Iteration: settings.Iteration,
				Error:     execErr,
			}
		}
	}

	if scenario.Transfer.Execute {
		payees := make([]*customerState, 0, len(scenario.Payees))
		for _, p := range scenario.Payees {
			// TODO introduce verification check
			payees = append(payees, customers[p])
		}

		execErr := r.doPayments(payer, payees, transferAmnts, settings)
		if execErr != nil {
			r.logger.Error(execErr)
			return &TestCaseResult{
				Name:      scenario.Name,
				Success:   false,
				Duration:  time.Since(start),
				Iteration: settings.Iteration,
				Error:     execErr,
			}
		}
	}

	duration := time.Since(start)
	r.logger.Infof("============= Finish test case %s, iter %d, duration: %ds =============", scenario.Name, settings.Iteration, duration)

	return &TestCaseResult{
		Name:      scenario.Name,
		Success:   true,
		Duration:  duration,
		Iteration: settings.Iteration,
	}
}

func (r *TestCaseRunner) doWithdrawals(customer *customerState, amounts []api.Amount, settings *TestCaseSettings) error {
	executorPool := pool.New().WithErrors().WithMaxGoroutines(settings.PoolSize)

	for _, amount := range amounts {
		time.Sleep(settings.CallsDelay)
		executorPool.Go(func() error {
			amount, err := r.intermediary.Withdraw(customer.Name, amount)
			if err != nil {
				return err
			}
			customer.AddWithdrawn(amount)
			return nil
		})
	}
	return executorPool.Wait()
}

func (r *TestCaseRunner) doPayments(payer *customerState, payees []*customerState, amounts []api.Amount, settings *TestCaseSettings) error {
	executorPool := pool.New().WithErrors().WithMaxGoroutines(settings.PoolSize)

	for i, amount := range amounts {
		payee := payees[i%len(payees)]
		time.Sleep(settings.CallsDelay)
		executorPool.Go(func() error {
			amount, err := r.intermediary.ExecutePayment(payer.Name, payee.Name, amount)
			if err != nil {
				return err
			}
			payer.AddPaidMount(amount)
			payee.AddReceivedMount(amount)
			return nil
		})
	}
	return executorPool.Wait()
}
