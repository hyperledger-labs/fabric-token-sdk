/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"

var (
	// ErrFailedToGetTMS is when an error occurs when getting an instance of a given TMS
	ErrFailedToGetTMS = errors.New("failed to get token manager")
)
