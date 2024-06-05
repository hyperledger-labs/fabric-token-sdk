/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type ObservableAuditorService struct {
	AuditService driver.AuditorService
	Metrics      *Metrics
}

func (o *ObservableAuditorService) AuditorCheck(request *driver.TokenRequest, metadata *driver.TokenRequestMetadata, anchor string) error {
	start := time.Now()
	err := o.AuditService.AuditorCheck(request, metadata, anchor)
	elapsed := time.Since(start)
	o.Metrics.ObserveAuditDuration(elapsed)
	o.Metrics.AddAudit(err == nil)
	return err
}
