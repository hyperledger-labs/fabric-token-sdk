/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"fmt"
	"strings"
	"time"
)

type retrievable interface {
	Avg() float64
	Min() float64
	Max() float64
}
type gettable interface {
	Get() float64
}

type reporter struct {
	*collector

	lastReportedTotalRequests uint64
}

func NewReporter(c *collector) Reporter {
	if c.supportsGetters {
		return &reporter{collector: c}
	}
	return &emptyReporter{}
}

const noValues = "No values can be retrieved. Change the provider type."

type emptyReporter struct{}

func (c *emptyReporter) GetTotalRequests() string  { return noValues }
func (c *emptyReporter) GetActiveRequests() string { return noValues }
func (c *emptyReporter) Summary() string           { return noValues }

func (c *reporter) GetTotalRequests() string {
	currentTotalRequests := uint64(c.TotalRequests.(gettable).Get())
	requestsSinceLastReport := currentTotalRequests - c.lastReportedTotalRequests
	c.lastReportedTotalRequests = currentTotalRequests
	return fmt.Sprintf("Total requests since last report: %d", requestsSinceLastReport)
}

func (c *reporter) GetActiveRequests() string {
	return fmt.Sprintf("Active requests: %d", int(c.ActiveRequests.(gettable).Get()))
}

func (c *reporter) Summary() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("Total withdraw requests %d, average duration of the request %v\n",
		int(c.TotalWithdrawRequests.(gettable).Get()),
		c.WithdrawDuration.(retrievable).Avg()))
	if c.TotalWithdrawRequests.(gettable).Get() != 0 {
		b.WriteString(fmt.Sprintf("Success ratio %.2f%%\n",
			c.TotalSuccessWithdrawRequests.(gettable).Get()/c.TotalWithdrawRequests.(gettable).Get()*100))
		b.WriteString(fmt.Sprintf("Minimum withdraw request took %v, maximum %v\n",
			time.Duration(c.WithdrawDuration.(retrievable).Min()),
			time.Duration(c.WithdrawDuration.(retrievable).Max())))
	}

	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Total payment requests %d, average duration of the request %v\n",
		int(c.TotalTransferRequests.(gettable).Get()),
		time.Duration(c.TransferDuration.(retrievable).Avg())))
	if c.TotalTransferRequests.(gettable).Get() != 0 {
		b.WriteString(fmt.Sprintf("Success ratio %.2f%%\n",
			c.TotalSuccessTransferRequests.(gettable).Get()/c.TotalTransferRequests.(gettable).Get()*100))
		b.WriteString(fmt.Sprintf("Minimum payment request took %v, maximum %v\n",
			time.Duration(c.TransferDuration.(retrievable).Min()),
			time.Duration(c.TransferDuration.(retrievable).Max())))
	}

	return b.String()
}
