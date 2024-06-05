/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type ObservableIssueService struct {
	IssueService driver.IssueService
	Metrics      *Metrics
}

func NewObservableIssueService(issueService driver.IssueService, metrics *Metrics) *ObservableIssueService {
	return &ObservableIssueService{IssueService: issueService, Metrics: metrics}
}

func (o *ObservableIssueService) Issue(issuerIdentity driver.Identity, tokenType string, values []uint64, owners [][]byte, opts *driver.IssueOptions) (driver.IssueAction, *driver.IssueMetadata, error) {
	start := time.Now()
	action, meta, err := o.IssueService.Issue(issuerIdentity, tokenType, values, owners, opts)
	duration := time.Since(start)
	o.Metrics.ObserveIssueDuration(duration)
	o.Metrics.AddIssue(tokenType, err == nil)
	return action, meta, err
}

func (o *ObservableIssueService) VerifyIssue(tr driver.IssueAction, metadata [][]byte) error {
	return o.IssueService.VerifyIssue(tr, metadata)
}

func (o *ObservableIssueService) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	return o.IssueService.DeserializeIssueAction(raw)
}
