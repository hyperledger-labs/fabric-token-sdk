/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package errs

import "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"

var (
	ErrLockContention     = errors.New("auditor enrollment id lock contention")
	ErrLockAcquireTimeout = errors.New("auditor enrollment id lock acquire timeout")
	ErrLockLost           = errors.New("auditor enrollment id lock lost")
	ErrLockNotHeld        = errors.New("auditor enrollment id locks not held")
)
